package provisioner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/boykaspaces/hermes-personal-tools/clients/credential-agent/internal/lease"
)

const maxResponseBytes = 1 << 20

type Config struct {
	Endpoint      string
	Region        string
	Profile       string
	OutputPath    string
	RefreshBefore time.Duration
}

type Credentials interface {
	Retrieve(context.Context) (aws.Credentials, error)
}

type Client struct {
	config      Config
	credentials Credentials
	signer      *v4.Signer
	httpClient  *http.Client
	now         func() time.Time
	logger      *slog.Logger
}

func New(config Config, credentials Credentials, logger *slog.Logger) (*Client, error) {
	parsed, err := url.Parse(config.Endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.Path != "/v1/credential-leases" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("credential endpoint must be an HTTPS /v1/credential-leases URL")
	}
	if config.Region == "" || config.Profile == "" || config.OutputPath == "" {
		return nil, errors.New("region, profile, and output path are required")
	}
	if config.RefreshBefore <= 0 {
		config.RefreshBefore = 10 * time.Minute
	}
	return &Client{
		config: config, credentials: credentials, signer: v4.NewSigner(),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now: time.Now, logger: logger,
	}, nil
}

func (c *Client) Refresh(ctx context.Context) (lease.Lease, error) {
	body, err := json.Marshal(map[string]string{"profile": c.config.Profile})
	if err != nil {
		return lease.Lease{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return lease.Lease{}, fmt.Errorf("create lease request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-store")
	credentials, err := c.credentials.Retrieve(ctx)
	if err != nil {
		return lease.Lease{}, fmt.Errorf("retrieve AWS credentials: %w", err)
	}
	digest := sha256.Sum256(body)
	if err := c.signer.SignHTTP(ctx, credentials, request, hex.EncodeToString(digest[:]), "execute-api", c.config.Region, c.now()); err != nil {
		return lease.Lease{}, fmt.Errorf("sign lease request: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return lease.Lease{}, fmt.Errorf("request credential lease: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxResponseBytes))
		return lease.Lease{}, fmt.Errorf("credential lease endpoint returned status %d", response.StatusCode)
	}
	var issued lease.Lease
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&issued); err != nil {
		return lease.Lease{}, fmt.Errorf("decode credential lease: %w", err)
	}
	if issued.Profile != c.config.Profile {
		return lease.Lease{}, errors.New("credential lease profile mismatch")
	}
	if err := issued.Validate(c.now()); err != nil {
		return lease.Lease{}, fmt.Errorf("validate credential lease: %w", err)
	}
	if err := writeAtomic(c.config.OutputPath, issued); err != nil {
		return lease.Lease{}, err
	}
	c.logger.Info("credential lease refreshed", "profile", issued.Profile, "provider", issued.Provider, "lease_id", issued.LeaseID, "expires_at", issued.ExpiresAt)
	return issued, nil
}

func (c *Client) Run(ctx context.Context) error {
	backoff := 5 * time.Second
	for {
		issued, err := c.Refresh(ctx)
		if err != nil {
			c.logger.Error("credential lease refresh failed", "profile", c.config.Profile, "error", err)
			if !wait(ctx, backoff) {
				return ctx.Err()
			}
			if backoff < time.Minute {
				backoff *= 2
			}
			continue
		}
		backoff = 5 * time.Second
		delay := issued.ExpiresAt.Sub(c.now()) - c.config.RefreshBefore
		if delay < time.Minute {
			delay = time.Minute
		}
		if !wait(ctx, delay) {
			return ctx.Err()
		}
	}
}

func writeAtomic(path string, value lease.Lease) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create credential directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("secure credential directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".lease-*")
	if err != nil {
		return fmt.Errorf("create temporary lease: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("secure temporary lease: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	if err := encoder.Encode(value); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode lease: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync lease: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close lease: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace lease: %w", err)
	}
	return nil
}

func wait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

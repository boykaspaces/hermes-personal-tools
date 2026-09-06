// Command google-oauth-bootstrap performs a one-time Google Calendar OAuth
// authorization and writes the resulting credential directly to AWS Secrets
// Manager. It never prints or persists the refresh token locally.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const (
	defaultRegion   = "ap-southeast-1"
	defaultSecretID = "personal-tools-google-workspace-oauth"
	oauthTimeout    = 5 * time.Minute
)

type options struct {
	credentialsPath string
	region          string
	secretID        string
}

type authorizationResult struct {
	code string
	err  error
}

type storedCredential struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	TokenURI     string `json:"token_uri"`
}

func main() {
	log.SetFlags(0)
	if err := run(context.Background(), parseOptions()); err != nil {
		log.Fatal(err)
	}
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.credentialsPath, "credentials", "", "path to a downloaded Google Desktop OAuth client JSON file")
	flag.StringVar(&opts.region, "region", defaultRegion, "AWS region containing the credential Secret")
	flag.StringVar(&opts.secretID, "secret-id", defaultSecretID, "AWS Secrets Manager Secret name or ARN")
	flag.Parse()
	if strings.TrimSpace(opts.credentialsPath) == "" {
		log.Fatal("--credentials is required")
	}
	return opts
}

func run(parent context.Context, opts options) error {
	clientJSON, err := os.ReadFile(opts.credentialsPath)
	if err != nil {
		return fmt.Errorf("read Google OAuth client JSON: %w", err)
	}
	oauthConfig, err := google.ConfigFromJSON(clientJSON, googlecalendar.CalendarReadonlyScope)
	if err != nil {
		return fmt.Errorf("parse Google Desktop OAuth client JSON: %w", err)
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start OAuth loopback listener: %w", err)
	}
	defer func() { _ = listener.Close() }()

	oauthConfig.RedirectURL = "http://" + listener.Addr().String() + "/oauth/callback"
	state, err := randomState()
	if err != nil {
		return err
	}
	verifier := oauth2.GenerateVerifier()
	result := make(chan authorizationResult, 1)
	server := oauthCallbackServer(listener, state, result)
	defer func() { _ = server.Close() }()

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			select {
			case result <- authorizationResult{err: fmt.Errorf("serve OAuth callback: %w", serveErr)}:
			default:
			}
		}
	}()

	authURL := oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	)
	fmt.Println("Open this Google authorization URL in your browser:")
	fmt.Println(authURL)
	fmt.Printf("Waiting up to %s for the local callback...\n", oauthTimeout)

	ctx, cancel := context.WithTimeout(parent, oauthTimeout)
	defer cancel()
	var authResult authorizationResult
	select {
	case authResult = <-result:
	case <-ctx.Done():
		return errors.New("timed out waiting for Google OAuth callback")
	}
	if authResult.err != nil {
		return authResult.err
	}

	token, err := oauthConfig.Exchange(ctx, authResult.code, oauth2.VerifierOption(verifier))
	if err != nil {
		return fmt.Errorf("exchange Google authorization code: %w", err)
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		return errors.New("google did not return a refresh token; revoke the app grant and retry with consent")
	}

	if err := verifyCalendarAccess(ctx, oauthConfig, token); err != nil {
		return err
	}
	if err := putCredentialSecret(ctx, opts, oauthConfig, token.RefreshToken); err != nil {
		return err
	}

	fmt.Println("calendar_readonly_access_ok=true")
	fmt.Println("google_credential_secret_updated=true")
	return nil
}

func oauthCallbackServer(listener net.Listener, expectedState string, result chan<- authorizationResult) *http.Server {
	var once sync.Once
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(response http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		callbackResult := authorizationResult{}
		switch {
		case query.Get("state") != expectedState:
			callbackResult.err = errors.New("google OAuth state mismatch")
		case query.Get("error") != "":
			callbackResult.err = fmt.Errorf("google OAuth authorization failed: %s", query.Get("error"))
		case query.Get("code") == "":
			callbackResult.err = errors.New("google OAuth callback did not include an authorization code")
		default:
			callbackResult.code = query.Get("code")
		}
		once.Do(func() { result <- callbackResult })

		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		message := "Authorization received. You can close this tab and return to Codex."
		if callbackResult.err != nil {
			message = "Authorization failed: " + callbackResult.err.Error()
		}
		_, _ = fmt.Fprintf(response, "<!doctype html><title>Personal Tools OAuth</title><p>%s</p>", html.EscapeString(message))
	})
	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func randomState() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate OAuth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func verifyCalendarAccess(ctx context.Context, oauthConfig *oauth2.Config, token *oauth2.Token) error {
	httpClient := oauthConfig.Client(ctx, token)
	service, err := googlecalendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("initialize Google Calendar verification client: %w", err)
	}
	_, err = service.Events.List("primary").
		Context(ctx).
		TimeMin(time.Now().Format(time.RFC3339)).
		MaxResults(1).
		SingleEvents(true).
		OrderBy("startTime").
		ShowDeleted(false).
		Do()
	if err != nil {
		return fmt.Errorf("verify Google Calendar read-only access: %w", err)
	}
	return nil
}

func putCredentialSecret(ctx context.Context, opts options, oauthConfig *oauth2.Config, refreshToken string) error {
	secretJSON, err := json.Marshal(storedCredential{
		ClientID:     oauthConfig.ClientID,
		ClientSecret: oauthConfig.ClientSecret,
		RefreshToken: refreshToken,
		TokenURI:     oauthConfig.Endpoint.TokenURL,
	})
	if err != nil {
		return fmt.Errorf("encode Google credential Secret: %w", err)
	}

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("load AWS configuration: %w", err)
	}
	_, err = secretsmanager.NewFromConfig(awsConfig).PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     &opts.secretID,
		SecretString: stringPointer(string(secretJSON)),
	})
	if err != nil {
		return fmt.Errorf("write Google credential to AWS Secrets Manager: %w", err)
	}
	return nil
}

func stringPointer(value string) *string { return &value }

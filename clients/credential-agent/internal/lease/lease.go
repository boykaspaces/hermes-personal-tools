package lease

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const CredentialHTTPBasic = "http-basic"

type Target struct {
	Protocol   string `json:"protocol"`
	Host       string `json:"host"`
	PathPrefix string `json:"path_prefix"`
}

type Credential struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Secret   string `json:"secret"`
}

type Lease struct {
	LeaseID    string     `json:"lease_id"`
	Profile    string     `json:"profile"`
	Provider   string     `json:"provider"`
	Credential Credential `json:"credential"`
	Targets    []Target   `json:"targets"`
	ExpiresAt  time.Time  `json:"expires_at"`
}

func (l Lease) Validate(now time.Time) error {
	if l.LeaseID == "" || l.Profile == "" || l.Provider == "" {
		return errors.New("lease metadata is incomplete")
	}
	if l.Credential.Type != CredentialHTTPBasic || l.Credential.Username == "" || l.Credential.Secret == "" {
		return errors.New("lease credential is incomplete or unsupported")
	}
	if !l.ExpiresAt.After(now.Add(30 * time.Second)) {
		return errors.New("lease is expired or too close to expiry")
	}
	if len(l.Targets) == 0 {
		return errors.New("lease has no targets")
	}
	for _, target := range l.Targets {
		if target.Protocol == "" || target.Host == "" || target.PathPrefix == "" {
			return errors.New("lease target is incomplete")
		}
	}
	return nil
}

func (l Lease) Matches(protocol, host, path string) bool {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	path = normalizePath(path)
	for _, target := range l.Targets {
		if protocol != strings.ToLower(target.Protocol) || host != strings.ToLower(strings.TrimSuffix(target.Host, ".")) {
			continue
		}
		targetPath := normalizePath(target.PathPrefix)
		if path == targetPath || strings.HasPrefix(path, targetPath+"/") {
			return true
		}
	}
	return false
}

func LoadDirectory(directory string, now time.Time) ([]Lease, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	leases := make([]Lease, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		value, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read lease %s: %w", entry.Name(), err)
		}
		var item Lease
		decoder := json.NewDecoder(strings.NewReader(string(value)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&item); err != nil {
			return nil, fmt.Errorf("decode lease %s: %w", entry.Name(), err)
		}
		if err := item.Validate(now); err != nil {
			continue
		}
		leases = append(leases, item)
	}
	return leases, nil
}

func normalizePath(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, ".git")
	return path
}

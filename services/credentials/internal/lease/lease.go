package lease

import "time"

const (
	CredentialHTTPBasic = "http-basic"
)

type Request struct {
	Profile string `json:"profile"`
}

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

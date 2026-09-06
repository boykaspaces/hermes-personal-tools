package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/lease"
)

type Issuer interface {
	Issue(context.Context) (lease.Lease, error)
}

type Registration struct {
	Profile string
	Issuer  Issuer
}

type Registry struct {
	issuers map[string]Issuer
}

func NewRegistry(registrations ...Registration) (*Registry, error) {
	registry := &Registry{issuers: make(map[string]Issuer, len(registrations))}
	for _, registration := range registrations {
		profile := strings.TrimSpace(registration.Profile)
		if profile == "" {
			return nil, errors.New("credential provider profile is required")
		}
		if registration.Issuer == nil {
			return nil, fmt.Errorf("credential provider %q has no issuer", profile)
		}
		if _, exists := registry.issuers[profile]; exists {
			return nil, fmt.Errorf("credential provider profile %q is registered more than once", profile)
		}
		registry.issuers[profile] = registration.Issuer
	}
	if len(registry.issuers) == 0 {
		return nil, errors.New("at least one credential provider is required")
	}
	return registry, nil
}

func (r *Registry) Lookup(profile string) (Issuer, bool) {
	if r == nil {
		return nil, false
	}
	issuer, ok := r.issuers[profile]
	return issuer, ok
}

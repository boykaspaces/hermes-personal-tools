package provider

import (
	"context"
	"testing"

	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/lease"
)

type fakeIssuer struct{}

func (fakeIssuer) Issue(context.Context) (lease.Lease, error) { return lease.Lease{}, nil }

func TestRegistryRoutesExactProfiles(t *testing.T) {
	first := fakeIssuer{}
	second := fakeIssuer{}
	registry, err := NewRegistry(
		Registration{Profile: "github-one", Issuer: first},
		Registration{Profile: "gitlab-two", Issuer: second},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Lookup("gitlab-two"); !ok {
		t.Fatal("registered profile was not found")
	}
	if _, ok := registry.Lookup("unknown"); ok {
		t.Fatal("unknown profile was found")
	}
}

func TestRegistryRejectsInvalidRegistrations(t *testing.T) {
	tests := []struct {
		name          string
		registrations []Registration
	}{
		{name: "empty"},
		{name: "blank profile", registrations: []Registration{{Profile: " ", Issuer: fakeIssuer{}}}},
		{name: "nil issuer", registrations: []Registration{{Profile: "github", Issuer: nil}}},
		{name: "duplicate profile", registrations: []Registration{
			{Profile: "github", Issuer: fakeIssuer{}},
			{Profile: "github", Issuer: fakeIssuer{}},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewRegistry(test.registrations...); err == nil {
				t.Fatal("NewRegistry accepted invalid registrations")
			}
		})
	}
}

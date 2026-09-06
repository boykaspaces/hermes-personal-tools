package auth

import (
	"context"
	"errors"
	"testing"
)

func TestValidateBearer(t *testing.T) {
	tests := []struct {
		name   string
		header string
		ok     bool
	}{
		{name: "valid", header: "Bearer local-secret", ok: true},
		{name: "case insensitive scheme", header: "bearer local-secret", ok: true},
		{name: "wrong token", header: "Bearer wrong"},
		{name: "missing scheme", header: "local-secret"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateBearer(context.Background(), test.header, StaticToken("local-secret"))
			if test.ok && err != nil {
				t.Fatalf("ValidateBearer() error = %v", err)
			}
			if !test.ok && !errors.Is(err, ErrInvalidBearerToken) {
				t.Fatalf("ValidateBearer() error = %v", err)
			}
		})
	}
}

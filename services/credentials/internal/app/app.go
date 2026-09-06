package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/lease"
	"github.com/boykaspaces/hermes-personal-tools/services/credentials/internal/provider"
)

const maxRequestBytes = 4096

func New(registry *provider.Registry, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/credential-leases", func(w http.ResponseWriter, r *http.Request) {
		setResponseHeaders(w)
		caller, ok := CallerFromContext(r.Context())
		if !ok || caller == "" {
			writeError(w, http.StatusUnauthorized, "authenticated caller identity is required")
			return
		}
		var request lease.Request
		decoder := json.NewDecoder(io.LimitReader(r.Body, maxRequestBytes))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON request")
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "request must contain one JSON object")
			return
		}
		issuer, ok := registry.Lookup(request.Profile)
		if !ok {
			writeError(w, http.StatusNotFound, "credential profile not found")
			return
		}
		issued, err := issuer.Issue(r.Context())
		if err != nil {
			logger.Error("credential lease issuance failed", "caller", caller, "profile", request.Profile, "error", err)
			writeError(w, http.StatusBadGateway, "credential provider failed")
			return
		}
		logger.Info("credential lease issued", "caller", caller, "profile", issued.Profile, "provider", issued.Provider, "lease_id", issued.LeaseID, "expires_at", issued.ExpiresAt)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(issued)
	})
	return mux
}

func setResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

type callerContextKey struct{}

func WithCaller(ctx context.Context, caller string) context.Context {
	return context.WithValue(ctx, callerContextKey{}, caller)
}

func CallerFromContext(ctx context.Context) (string, bool) {
	caller, ok := ctx.Value(callerContextKey{}).(string)
	return caller, ok
}

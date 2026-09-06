package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/app"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/audit"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/catalog"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/config"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/identity"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/secrets"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/transport"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	token := os.Getenv("GATEWAY_BEARER_TOKEN")
	if token == "" {
		logger.Error("GATEWAY_BEARER_TOKEN is required")
		os.Exit(1)
	}
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = "127.0.0.1:8080"
	}

	cfg, err := config.FromEnvironment(os.Getenv)
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}
	if cfg.UsesLiveProviders() {
		logger.Error("local server supports only disabled and mock provider modes")
		os.Exit(1)
	}
	recorder := audit.New(logger)
	credentials := secrets.NewCachedStringSource(nil)
	configuredFeatures, err := features.Build(context.Background(), catalog.Definitions(cfg, credentials, recorder))
	if err != nil {
		logger.Error("initialize features", "error", err)
		os.Exit(1)
	}
	application := app.New(configuredFeatures, logger)
	handler := localAuthentication(application, token, logger)
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	logger.Info("local_server_started", "address", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("local_server_failed", "error", err)
		os.Exit(1)
	}
}

// localAuthentication simulates the production Authorizer for local MCP
// protocol tests. It is not compiled into the MCP Lambda binary.
func localAuthentication(next http.Handler, expectedToken string, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if !validLocalBearer(request.Header.Get(transport.ClientTokenHeader), expectedToken) {
			logger.WarnContext(request.Context(), "authorization_denied")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		request.Header.Del(transport.ClientTokenHeader)
		request.Header.Del("Authorization")
		ctx := identity.WithCaller(request.Context(), identity.New("hermes"))
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}

func validLocalBearer(value, expected string) bool {
	const prefix = "Bearer "
	if len(value) <= len(prefix) || value[:len(prefix)] != prefix {
		return false
	}
	presented := value[len(prefix):]
	return len(presented) == len(expected) && subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) == 1
}

package app

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/features"
	"github.com/boykaspaces/hermes-personal-tools/services/mcp/internal/protocol"
)

func New(configuredFeatures []features.Feature, logger *slog.Logger) http.Handler {
	modules := make([]protocol.Module, 0, len(configuredFeatures))
	statuses := make(map[string]string, len(configuredFeatures))
	for _, feature := range configuredFeatures {
		statuses[feature.Name] = feature.Mode
		if feature.Module != nil {
			modules = append(modules, feature.Module)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("POST /mcp", protocol.NewHandler(modules, logger))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Status   string            `json:"status"`
			Features map[string]string `json:"features"`
		}{
			Status:   "ok",
			Features: statuses,
		})
	})
	return mux
}

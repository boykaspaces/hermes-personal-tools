package protocol

import (
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Module owns one bounded group of MCP tools, such as Calendar, Gmail or AWS operations.
// Disabled features are omitted from the module list and therefore cannot
// appear in tools/list or be invoked by name.
type Module interface {
	Register(*mcp.Server)
}

func NewHandler(modules []Module, logger *slog.Logger) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "personal-tools",
		Version: "0.10.1",
	}, nil)
	for _, module := range modules {
		module.Register(server)
	}

	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		Logger:                       logger,
		PropagateRequestCancellation: true,
	})
}

package audit

import (
	"context"
	"log/slog"
	"time"
)

type Recorder struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Recorder {
	return &Recorder{logger: logger}
}

// ToolCall records only a bounded summary. Raw arguments, bearer tokens and
// provider credentials must never be passed to this method.
func (r *Recorder) ToolCall(ctx context.Context, callerID, tool, outcome string, started time.Time, resultCount int) {
	r.logger.InfoContext(ctx, "tool_call",
		"caller_id", callerID,
		"tool", tool,
		"outcome", outcome,
		"duration_ms", time.Since(started).Milliseconds(),
		"result_count", resultCount,
	)
}

package start

import (
	"context"
	"log/slog"
	"os"
)

const uuidKey = "uuid"

type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Check if the UUID exists in the context
	if v, ok := ctx.Value(uuidKey).(string); ok {
		// Add the attribute to the record before passing it down
		r.AddAttrs(slog.String("uuid", v))
	}
	return h.Handler.Handle(ctx, r)
}

func configureLogger() {
	baseHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(&ContextHandler{baseHandler})

	// Set as global logger
	slog.SetDefault(logger)
}

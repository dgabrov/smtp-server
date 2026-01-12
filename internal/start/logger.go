package start

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/dgb9/smtp-server/internal/data"
	"gopkg.in/natefinch/lumberjack.v2"
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

func configureLogger(c data.ConfigData) {
	logConfig := c.Log

	lumberjackLogger := &lumberjack.Logger{
		Filename:   logConfig.Filename,
		MaxSize:    logConfig.MaxSize,    // megabytes before rotation
		MaxBackups: logConfig.MaxBackups, // max number of old log files to keep
		MaxAge:     logConfig.MaxAge,     // max days to retain old log files
		Compress:   logConfig.Compress,   // whether to compress (gzip) old log files
	}

	writer := io.MultiWriter(os.Stdout, lumberjackLogger)

	baseHandler := slog.NewJSONHandler(writer, nil)
	logger := slog.New(&ContextHandler{baseHandler})

	// Set as global logger
	slog.SetDefault(logger)
}

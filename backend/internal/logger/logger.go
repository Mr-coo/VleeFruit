// Package logger builds the application's structured logger (log/slog) and
// carries request-scoped fields through context.
//
// Format is environment-driven: development uses a human-readable text handler,
// production emits JSON for log aggregators. The level (debug|info|warn|error)
// is configurable so noisy levels can be silenced in production.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

var requestIDKey = ctxKey{}

// WithRequestID returns a context carrying the request id, so any log record
// made with that context is automatically tagged with request_id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext extracts a request id stored by WithRequestID.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}

// New builds a leveled slog.Logger. env selects the handler (production => JSON,
// otherwise text); level is one of debug|info|warn|error (default info).
func New(env, level string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var base slog.Handler
	if IsProduction(env) {
		base = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		base = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(&contextHandler{Handler: base})
}

// IsProduction reports whether env names a production-like environment.
func IsProduction(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// contextHandler decorates every record with the request id from its context,
// so handlers and services do not have to pass it explicitly.
type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := RequestIDFromContext(ctx); ok {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}

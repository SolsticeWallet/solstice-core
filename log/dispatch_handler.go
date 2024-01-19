package log

import (
	"context"
	"log/slog"
)

// DispatchHandler dispatches the incomming log record to the configured
// handlers.
type DispatchHandler struct {
	handlers []slog.Handler
}

// NewDispatchHandler creates a new dispatchHandler.
func NewDispatchHandler(handlers ...slog.Handler) slog.Handler {
	return &DispatchHandler{
		handlers: handlers,
	}
}

// Handle implements slog.Handler.
func (h *DispatchHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// Enabled implements slog.Handler.
// As soon as one of the child handlers is enabled for the log level, true is
// returned; otherwise false.
func (h *DispatchHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

// WithGroup implements slog.Handler.
func (h *DispatchHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithGroup(name))
	}
	return NewDispatchHandler(handlers...)
}

// WithAttrs implements slog.Handler.
func (h *DispatchHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithAttrs(attrs))
	}
	return NewDispatchHandler(handlers...)
}

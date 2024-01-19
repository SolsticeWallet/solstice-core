package log

import (
	"log/slog"
	"sync/atomic"
)

var root atomic.Value

func CreateDefaultLogger(logdir string, name string) {
	defaultLogger := newDefaultLogger(logdir, name)
	root.Store(defaultLogger)
	slog.SetDefault(defaultLogger)
}

/*
func Root() *slog.Logger {
	return root.Load().(*slog.Logger)
}

func Debug(msg string, args ...any) {
	Root().Debug(msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	Root().DebugContext(ctx, msg, args...)
}

func Info(msg string, args ...any) {
	Root().Info(msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	Root().InfoContext(ctx, msg, args...)
}

func Warn(msg string, args ...any) {
	Root().Warn(msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	Root().WarnContext(ctx, msg, args...)
}

func Error(msg string, args ...any) {
	Root().Error(msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	Root().ErrorContext(ctx, msg, args...)
}
*/

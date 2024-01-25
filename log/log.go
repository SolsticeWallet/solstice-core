package log

import (
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/google/uuid"
)

var root atomic.Value

func CreateDefaultLogger(logdir string, name string) {
	defaultLogger := newDefaultLogger(logdir, name)
	root.Store(defaultLogger)
	slog.SetDefault(defaultLogger)
}

func CreateDebugLogger(logdir string, name string) {
	defaultLogger := newLogger(
		logdir, name,
		slog.LevelDebug, slog.LevelDebug,
		os.Stdout,
	)
	root.Store(defaultLogger)
	slog.SetDefault(defaultLogger)
}

// Subscribe subscribes to log events. The key identifies the subscription
// and can be used to unsubscribe at a later stage.
func Subscribe(level slog.Level, chnl chan slog.Record) (key string) {
	key = uuid.NewString()
	chnlHandler.Subscribe(key, level, chnl)
	return
}

// Unsubscribe from a log event by giving the subscription key.
func Unsubscribe(key string) {
	chnlHandler.Unsubscribe(key)
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

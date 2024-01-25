package log

import (
	"log/slog"
	"os"
)

var chnlHandler *ChannelHandler

func init() {
	chnlHandler = NewChannelHandler().(*ChannelHandler)
}

// newDefaultLogger Create a new logger with the given log directory and
// log name.
func newDefaultLogger(logdir string, logname string) *slog.Logger {
	return newLogger(
		logdir, logname,
		slog.LevelDebug, slog.LevelInfo,
		os.Stdout,
	)
}

// newLogger Creates a new logger with the given log directory, log name,
// terminal log level, file log level and terminal output file handle.
func newLogger(
	logdir string,
	logname string,
	termLevel, fileLevel slog.Level,
	termIO *os.File,
) *slog.Logger {
	handlers := make([]slog.Handler, 0)
	if termIO != nil {
		handlers = append(
			handlers,
			slog.NewTextHandler(
				termIO,
				&slog.HandlerOptions{
					AddSource: true,
					Level:     termLevel,
				},
			),
		)
	}
	handlers = append(
		handlers,
		NewFileHandlerWithLevel(logdir, logname, fileLevel),
	)
	handlers = append(
		handlers,
		chnlHandler,
	)

	return slog.New(NewDispatchHandler(handlers...))
}

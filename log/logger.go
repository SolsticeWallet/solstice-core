package log

import (
	"log/slog"
	"os"
)

var chnlHandler *ChannelHandler

func init() {
	chnlHandler = NewChannelHandler().(*ChannelHandler)
}

func newDefaultLogger(logdir string, logname string) *slog.Logger {
	return newLogger(
		logdir, logname,
		slog.LevelDebug, slog.LevelInfo,
		os.Stdout,
	)
}

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

func Subscribe(key string, level slog.Level, chnl chan slog.Record) {
	chnlHandler.Subscribe(key, level, chnl)
}

func Unsubscribe(key string) {
	chnlHandler.Unsubscribe(key)
}

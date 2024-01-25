package log

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type channelSubscription struct {
	// The log level to subscribe to
	level slog.Level
	// The channel to send log records to
	chnl chan slog.Record
}

type ChannelHandler struct {
	// Map of subscriptions to channels
	subscriptions *sync.Map
	// List of groups
	groups []string
	// List of attributes
	attrs []slog.Attr
}

// NewChannelHandler creates a new ChannelHandler struct and returns it.
func NewChannelHandler() slog.Handler {
	return &ChannelHandler{
		subscriptions: &sync.Map{},
	}
}

// Handle forwards the log record on the subcribed channels.
func (h *ChannelHandler) Handle(ctx context.Context, r slog.Record) error {
	rec := r.Clone()
	rec.AddAttrs(h.attrs...)

	h.subscriptions.Range(func(k, v any) bool {
		subscr := v.(channelSubscription)
		if r.Level < subscr.level {
			return true
		}

		go func(chnl chan slog.Record, r slog.Record) {
			select {
			case chnl <- r:
			case <-time.After(time.Millisecond * 20):
			}
		}(subscr.chnl, rec)
		return true
	})
	return nil
}

// Enabled checks if the channel handler is enabled for a given log level
func (h *ChannelHandler) Enabled(_ context.Context, l slog.Level) bool {
	ok := false
	h.subscriptions.Range(func(k, v any) bool {
		subscr := v.(channelSubscription)
		if l >= subscr.level {
			ok = true
			return false
		}
		return true
	})
	return ok
}

// WithGroup creates a new handler with the given group name
func (h *ChannelHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	groups := make([]string, 0, len(h.groups)+1)
	if len(h.groups) > 0 {
		groups = append(groups, h.groups...)
	}
	groups = append(groups, name)

	return &ChannelHandler{
		subscriptions: h.subscriptions,
		groups:        groups,
		attrs:         h.attrs[:],
	}
}

// WithAttrs takes a slice of Attrs as parameters and returns a Handler.
func (h *ChannelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	attrList := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	if len(h.attrs) > 0 {
		attrList = append(attrList, h.attrs...)
	}
	if len(h.groups) > 0 {
		for _, attr := range attrs {
			key := strings.Join(h.groups, ".")
			key = strings.Join([]string{key, attr.Key}, ".")
			attr.Key = key
			attrList = append(attrList, attr)
		}
	} else {
		attrList = append(attrList, attrs...)
	}

	return &ChannelHandler{
		subscriptions: h.subscriptions,
		groups:        h.groups[:],
		attrs:         attrList,
	}
}

// Subscribe is used to subscribe to a channel. It takes a string key,
// a slog.Level level, and a channel of slog.Records as parameters.
func (h *ChannelHandler) Subscribe(
	key string,
	level slog.Level,
	chnl chan slog.Record,
) {
	h.subscriptions.Store(key, channelSubscription{
		level: level,
		chnl:  chnl,
	})
}

// Unsubscribe is used to unsubscribe a key from the channel handler
func (h *ChannelHandler) Unsubscribe(key string) {
	h.subscriptions.Delete(key)
}

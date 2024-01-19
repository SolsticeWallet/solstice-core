package log

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type channelSubscription struct {
	level slog.Level
	chnl  chan slog.Record
}

type ChannelHandler struct {
	subscriptions *sync.Map
	groups        []string
	attrs         []slog.Attr
}

func NewChannelHandler() slog.Handler {
	return &ChannelHandler{
		subscriptions: &sync.Map{},
	}
}

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

func (h *ChannelHandler) Unsubscribe(key string) {
	h.subscriptions.Delete(key)
}

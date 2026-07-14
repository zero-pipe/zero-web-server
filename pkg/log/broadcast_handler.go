package log

import (
	"context"
	"log/slog"
)

// BroadcastHandler wraps a slog.Handler and pushes formatted lines to Hub.
type BroadcastHandler struct {
	next  slog.Handler
	hub   *Hub
	attrs []slog.Attr
}

func NewBroadcastHandler(next slog.Handler, hub *Hub) *BroadcastHandler {
	if hub == nil {
		hub = DefaultHub
	}
	return &BroadcastHandler{next: next, hub: hub}
}

func (h *BroadcastHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *BroadcastHandler) Handle(ctx context.Context, r slog.Record) error {
	err := h.next.Handle(ctx, r)
	if h.hub != nil && h.hub.ClientCount() > 0 {
		if len(h.attrs) > 0 {
			r.AddAttrs(h.attrs...)
		}
		h.hub.Broadcast(FormatRecordText(r))
	}
	return err
}

func (h *BroadcastHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &BroadcastHandler{
		next:  h.next.WithAttrs(attrs),
		hub:   h.hub,
		attrs: append(append([]slog.Attr{}, h.attrs...), attrs...),
	}
}

func (h *BroadcastHandler) WithGroup(name string) slog.Handler {
	return &BroadcastHandler{
		next:  h.next.WithGroup(name),
		hub:   h.hub,
		attrs: h.attrs,
	}
}

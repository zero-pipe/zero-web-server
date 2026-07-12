package log

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// BroadcastHandler wraps a slog.Handler and pushes formatted lines to Hub.
type BroadcastHandler struct {
	next slog.Handler
	hub  *Hub
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
		h.hub.Broadcast(formatRecord(r))
	}
	return err
}

func (h *BroadcastHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &BroadcastHandler{next: h.next.WithAttrs(attrs), hub: h.hub}
}

func (h *BroadcastHandler) WithGroup(name string) slog.Handler {
	return &BroadcastHandler{next: h.next.WithGroup(name), hub: h.hub}
}

func formatRecord(r slog.Record) string {
	var b strings.Builder
	b.WriteString(r.Time.Local().Format("2006-01-02 15:04:05.000"))
	b.WriteByte(' ')
	b.WriteString(strings.ToUpper(r.Level.String()))
	b.WriteByte(' ')
	b.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		b.WriteByte(' ')
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(formatAttrValue(a.Value))
		return true
	})
	return b.String()
}

func formatAttrValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	default:
		return fmt.Sprint(v.Any())
	}
}

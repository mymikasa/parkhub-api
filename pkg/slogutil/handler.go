package slogutil

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type TraceContextHandler struct {
	next slog.Handler
}

func NewTraceContextHandler(next slog.Handler) *TraceContextHandler {
	return &TraceContextHandler{next: next}
}

func (h *TraceContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *TraceContextHandler) Handle(ctx context.Context, record slog.Record) error {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", spanContext.TraceID().String()),
			slog.String("span_id", spanContext.SpanID().String()),
		)
	}
	return h.next.Handle(ctx, record)
}

func (h *TraceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceContextHandler{next: h.next.WithAttrs(attrs)}
}

func (h *TraceContextHandler) WithGroup(name string) slog.Handler {
	return &TraceContextHandler{next: h.next.WithGroup(name)}
}

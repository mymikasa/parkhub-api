package slogutil

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestTraceContextHandlerAddsTraceFields(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewLogger(&buf, Options{Level: "info", Format: "json"})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	traceID, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	spanID, _ := trace.SpanIDFromHex("0123456789abcdef")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))

	logger.InfoContext(ctx, "hello")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := record["trace_id"]; got != traceID.String() {
		t.Fatalf("trace_id = %v, want %s", got, traceID.String())
	}
	if got := record["span_id"]; got != spanID.String() {
		t.Fatalf("span_id = %v, want %s", got, spanID.String())
	}
}

func TestIntoAndFromContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	ctx := IntoContext(context.Background(), logger)

	if got := FromContext(ctx); got != logger {
		t.Fatalf("FromContext() returned unexpected logger")
	}
}

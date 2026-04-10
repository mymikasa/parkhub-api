package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/parkhub/api/pkg/slogutil"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryLoggingInterceptorSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slogutil.NewTraceContextHandler(slog.NewJSONHandler(&buf, nil)))

	traceID, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	spanID, _ := trace.SpanIDFromHex("0123456789abcdef")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))

	interceptor := UnaryLoggingInterceptor(logger)
	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{FullMethod: "/identity.v1.TenantService/GetTenant"}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("UnaryLoggingInterceptor() error = %v", err)
	}

	records := decodeLogRecords(t, buf.String())
	if len(records) != 2 {
		t.Fatalf("got %d log records, want 2", len(records))
	}

	if got := records[0]["msg"]; got != "grpc request started" {
		t.Fatalf("started msg = %v", got)
	}
	if got := records[1]["msg"]; got != "grpc request finished" {
		t.Fatalf("finished msg = %v", got)
	}
	if got := records[1]["grpc_code"]; got != grpcCodes.OK.String() {
		t.Fatalf("grpc_code = %v, want %s", got, grpcCodes.OK.String())
	}
	if got := records[1]["trace_id"]; got != traceID.String() {
		t.Fatalf("trace_id = %v, want %s", got, traceID.String())
	}
	if got := records[1]["span_id"]; got != spanID.String() {
		t.Fatalf("span_id = %v, want %s", got, spanID.String())
	}
}

func TestUnaryLoggingInterceptorError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slogutil.NewTraceContextHandler(slog.NewJSONHandler(&buf, nil)))

	interceptor := UnaryLoggingInterceptor(logger)
	wantErr := status.Error(grpcCodes.InvalidArgument, "bad request")
	_, err := interceptor(context.Background(), "req", &grpc.UnaryServerInfo{FullMethod: "/identity.v1.TenantService/GetTenant"}, func(ctx context.Context, req any) (any, error) {
		return nil, wantErr
	})
	if status.Code(err) != grpcCodes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), grpcCodes.InvalidArgument)
	}

	records := decodeLogRecords(t, buf.String())
	if len(records) != 2 {
		t.Fatalf("got %d log records, want 2", len(records))
	}

	if got := records[1]["level"]; got != "ERROR" {
		t.Fatalf("level = %v, want ERROR", got)
	}
	if got := records[1]["grpc_code"]; got != grpcCodes.InvalidArgument.String() {
		t.Fatalf("grpc_code = %v, want %s", got, grpcCodes.InvalidArgument.String())
	}
	if got := records[1]["error"]; got != wantErr.Error() {
		t.Fatalf("error = %v, want %s", got, wantErr.Error())
	}
}

func TestUnaryRecoveryInterceptor(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slogutil.NewTraceContextHandler(slog.NewJSONHandler(&buf, nil)))

	recorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tracer := tracerProvider.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "request")

	interceptor := UnaryRecoveryInterceptor(logger)
	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{FullMethod: "/identity.v1.TenantService/GetTenant"}, func(ctx context.Context, req any) (any, error) {
		panic("boom")
	})
	span.End()

	if status.Code(err) != grpcCodes.Internal {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), grpcCodes.Internal)
	}

	records := decodeLogRecords(t, buf.String())
	if len(records) != 1 {
		t.Fatalf("got %d log records, want 1", len(records))
	}
	if got := records[0]["msg"]; got != "grpc panic recovered" {
		t.Fatalf("msg = %v, want grpc panic recovered", got)
	}
	if got := records[0]["panic"]; got != "boom" {
		t.Fatalf("panic = %v, want boom", got)
	}
	if _, ok := records[0]["stack_trace"]; !ok {
		t.Fatalf("stack_trace missing from log record")
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	if got := ended[0].Status().Code; got != codes.Error {
		t.Fatalf("span status = %v, want %v", got, codes.Error)
	}
}

func decodeLogRecords(t *testing.T, output string) []map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		record := map[string]any{}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("json.Unmarshal(%q) error = %v", line, err)
		}
		records = append(records, record)
	}
	return records
}

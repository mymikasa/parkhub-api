package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	runtimeotel "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName   string
	Environment   string
	TraceEndpoint string
	TraceInsecure bool
	SamplingRatio float64
}

type Providers struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	MetricsHandler http.Handler

	shutdown func(context.Context) error
}

func Init(ctx context.Context, cfg Config) (*Providers, error) {
	resource, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
		resource.WithTelemetrySDK(),
		resource.WithProcess(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	traceExporter, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(normalizeSamplingRatio(cfg.SamplingRatio)))),
	)

	registry := promclient.NewRegistry()
	metricExporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
		otelprom.WithoutScopeInfo(),
	)
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricExporter),
		sdkmetric.WithResource(resource),
	)

	if err := runtimeotel.Start(runtimeotel.WithMeterProvider(meterProvider)); err != nil {
		return nil, fmt.Errorf("start runtime metrics: %w", err)
	}

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return &Providers{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		MetricsHandler: promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		shutdown: func(ctx context.Context) error {
			return errors.Join(
				meterProvider.Shutdown(ctx),
				tracerProvider.Shutdown(ctx),
			)
		},
	}, nil
}

func (p *Providers) Shutdown(ctx context.Context) error {
	if p == nil || p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx)
}

func newTraceExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.TraceEndpoint),
	}
	if cfg.TraceInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}
	return exporter, nil
}

func normalizeSamplingRatio(ratio float64) float64 {
	switch {
	case ratio <= 0:
		return 0
	case ratio >= 1:
		return 1
	default:
		return ratio
	}
}

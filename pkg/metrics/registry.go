package metrics

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/metric"
)

var (
	globalMu sync.RWMutex
	global   *Registry
)

type Registry struct {
	meter metric.Meter
}

func Init(mp metric.MeterProvider) {
	globalMu.Lock()
	defer globalMu.Unlock()
	global = &Registry{meter: mp.Meter("parkhub")}
}

func Global() *Registry {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

func (r *Registry) Counter(name, description, unit string) (metric.Int64Counter, error) {
	return r.meter.Int64Counter(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
}

func (r *Registry) FloatCounter(name, description, unit string) (metric.Float64Counter, error) {
	return r.meter.Float64Counter(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
}

func (r *Registry) Histogram(name, description, unit string, boundaries []float64) (metric.Float64Histogram, error) {
	opts := []metric.Float64HistogramOption{
		metric.WithDescription(description),
		metric.WithUnit(unit),
	}
	if len(boundaries) > 0 {
		opts = append(opts, metric.WithExplicitBucketBoundaries(boundaries...))
	}
	return r.meter.Float64Histogram(name, opts...)
}

func (r *Registry) Gauge(name, description, unit string) (metric.Int64Gauge, error) {
	return r.meter.Int64Gauge(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
}

func (r *Registry) FloatGauge(name, description, unit string, f func(context.Context, metric.Float64Observer) error) error {
	_, err := r.meter.Float64ObservableGauge(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
		metric.WithFloat64Callback(f),
	)
	return err
}

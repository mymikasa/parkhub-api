package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func newTestProvider(t *testing.T) metric.MeterProvider {
	t.Helper()
	exp, err := prometheus.New()
	require.NoError(t, err)
	return sdkmetric.NewMeterProvider(sdkmetric.WithReader(exp))
}

func TestInit_SetsGlobal(t *testing.T) {
	mp := newTestProvider(t)
	Init(mp)
	reg := Global()
	require.NotNil(t, reg)
}

func TestCounter(t *testing.T) {
	mp := newTestProvider(t)
	Init(mp)
	reg := Global()
	require.NotNil(t, reg)

	ctr, err := reg.Counter("test_counter", "test counter", "1")
	require.NoError(t, err)
	assert.NotNil(t, ctr)

	ctr.Add(context.Background(), 1)
}

func TestHistogram(t *testing.T) {
	mp := newTestProvider(t)
	Init(mp)
	reg := Global()
	require.NotNil(t, reg)

	h, err := reg.Histogram("test_histogram", "test histogram", "ms", []float64{10, 50, 100, 500})
	require.NoError(t, err)
	assert.NotNil(t, h)

	h.Record(context.Background(), 42)
}

func TestGauge(t *testing.T) {
	mp := newTestProvider(t)
	Init(mp)
	reg := Global()
	require.NotNil(t, reg)

	g, err := reg.Gauge("test_gauge", "test gauge", "1")
	require.NoError(t, err)
	assert.NotNil(t, g)

	g.Record(context.Background(), 7)
}

func TestFloatGauge(t *testing.T) {
	mp := newTestProvider(t)
	Init(mp)
	reg := Global()
	require.NotNil(t, reg)

	err := reg.FloatGauge("test_float_gauge", "test float gauge", "1", func(_ context.Context, obs metric.Float64Observer) error {
		obs.Observe(3.14)
		return nil
	})
	require.NoError(t, err)
}

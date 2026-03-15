package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/metrics"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestTracing(t *testing.T) (*tracingManager, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	cfg := DefaultConfig()
	return newTracingManager(tp.Tracer("k6-test"), cfg), exporter
}

func makeTagSet(tags map[string]string) *metrics.TagSet {
	r := metrics.NewRegistry()
	ts := r.RootTagSet()
	for k, v := range tags {
		ts = ts.With(k, v)
	}
	return ts
}

func makeSample(name string, typ metrics.MetricType, tags map[string]string, value float64) metrics.Sample {
	r := metrics.NewRegistry()
	m := r.MustNewMetric(name, typ)
	ts := makeTagSet(tags)
	return metrics.Sample{
		TimeSeries: metrics.TimeSeries{
			Metric: m,
			Tags:   ts,
		},
		Value: value,
	}
}

func TestTracingManager_HTTPSpan(t *testing.T) {
	tm, exporter := setupTestTracing(t)

	sample := makeSample("http_reqs", metrics.Counter, map[string]string{
		"method": "GET", "url": "http://frontend:8080/api/products",
		"name": "/api/products", "status": "200", "vu": "1", "iter": "0",
	}, 1)

	tm.recordSample(context.Background(), sample)
	tm.endAllSpans()

	spans := exporter.GetSpans()
	require.GreaterOrEqual(t, len(spans), 1)

	found := false
	for _, s := range spans {
		if s.Name == "HTTP GET /api/products" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find HTTP span")
}

func TestTracingManager_CheckSpan_Pass(t *testing.T) {
	tm, exporter := setupTestTracing(t)

	sample := makeSample("checks", metrics.Rate, map[string]string{
		"check": "status is 200", "vu": "1", "iter": "0",
	}, 1)

	tm.recordSample(context.Background(), sample)
	tm.endAllSpans()

	spans := exporter.GetSpans()
	found := false
	for _, s := range spans {
		if s.Name == "check: status is 200" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find check span")
}

func TestTracingManager_CheckSpan_Fail(t *testing.T) {
	tm, exporter := setupTestTracing(t)

	sample := makeSample("checks", metrics.Rate, map[string]string{
		"check": "status is 200", "vu": "2", "iter": "0",
	}, 0) // failed

	tm.recordSample(context.Background(), sample)
	tm.endAllSpans()

	spans := exporter.GetSpans()
	found := false
	for _, s := range spans {
		if s.Name == "check: status is 200" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find failed check span")
}

func TestTracingManager_IterationLifecycle(t *testing.T) {
	tm, exporter := setupTestTracing(t)

	// HTTP request creates an iteration span
	httpSample := makeSample("http_reqs", metrics.Counter, map[string]string{
		"method": "GET", "url": "http://localhost/test",
		"name": "/test", "status": "200", "vu": "1", "iter": "0", "scenario": "default",
	}, 1)
	tm.recordSample(context.Background(), httpSample)

	// End iteration
	iterSample := makeSample("iterations", metrics.Counter, map[string]string{
		"vu": "1", "iter": "0",
	}, 1)
	tm.recordSample(context.Background(), iterSample)

	spans := exporter.GetSpans()
	require.GreaterOrEqual(t, len(spans), 2, "should have iteration + HTTP spans")

	var iterFound bool
	for _, s := range spans {
		if s.Name == "k6.iteration" {
			iterFound = true
		}
	}
	assert.True(t, iterFound, "should find iteration span")
}

func TestTracingManager_EndAllSpans(t *testing.T) {
	tm, exporter := setupTestTracing(t)

	httpSample := makeSample("http_reqs", metrics.Counter, map[string]string{
		"method": "GET", "url": "http://localhost/test",
		"name": "/test", "status": "200", "vu": "1", "iter": "0",
	}, 1)
	tm.recordSample(context.Background(), httpSample)

	// Don't send iterations metric — use endAllSpans
	tm.endAllSpans()

	spans := exporter.GetSpans()
	require.GreaterOrEqual(t, len(spans), 1)
}

func TestExtractTags(t *testing.T) {
	ts := makeTagSet(map[string]string{
		"method": "POST",
		"url":    "http://example.com",
		"status": "201",
	})
	tags := extractTags(ts)
	assert.Equal(t, "POST", tags["method"])
	assert.Equal(t, "http://example.com", tags["url"])
	assert.Equal(t, "201", tags["status"])
}

package otel

import (
	"context"
	"fmt"
	"sync"

	"github.com/mstoykov/atlas"
	"go.k6.io/k6/metrics"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// metricsRegistry tracks created OTel metric instruments, creating them lazily.
type metricsRegistry struct {
	meter      otelmetric.Meter
	prefix     string
	counters   sync.Map
	gauges     sync.Map
	histograms sync.Map
	rates      sync.Map
}

func newMetricsRegistry(meter otelmetric.Meter, prefix string) *metricsRegistry {
	return &metricsRegistry{meter: meter, prefix: prefix}
}

func (r *metricsRegistry) record(ctx context.Context, sample metrics.Sample) error {
	name := r.prefix + sample.Metric.Name
	attrs := tagsToAttributes(sample.Tags)
	opt := otelmetric.WithAttributeSet(attrs)

	switch sample.Metric.Type {
	case metrics.Counter:
		c, err := r.getOrCreateCounter(name, unitFor(sample.Metric.Contains))
		if err != nil {
			return err
		}
		c.Add(ctx, sample.Value, opt)

	case metrics.Gauge:
		g, err := r.getOrCreateGauge(name, unitFor(sample.Metric.Contains))
		if err != nil {
			return err
		}
		g.Record(ctx, sample.Value, opt)

	case metrics.Trend:
		h, err := r.getOrCreateHistogram(name, unitFor(sample.Metric.Contains))
		if err != nil {
			return err
		}
		h.Record(ctx, sample.Value, opt)

	case metrics.Rate:
		nonZero, total, err := r.getOrCreateRate(name)
		if err != nil {
			return err
		}
		if sample.Value != 0 {
			nonZero.Add(ctx, 1, opt)
		}
		total.Add(ctx, 1, opt)
	}

	return nil
}

func (r *metricsRegistry) getOrCreateCounter(name, unit string) (otelmetric.Float64Counter, error) {
	if v, ok := r.counters.Load(name); ok {
		return v.(otelmetric.Float64Counter), nil
	}
	var opts []otelmetric.Float64CounterOption
	if unit != "" {
		opts = append(opts, otelmetric.WithUnit(unit))
	}
	c, err := r.meter.Float64Counter(name, opts...)
	if err != nil {
		return nil, fmt.Errorf("create counter %q: %w", name, err)
	}
	r.counters.Store(name, c)
	return c, nil
}

func (r *metricsRegistry) getOrCreateGauge(name, unit string) (otelmetric.Float64Gauge, error) {
	if v, ok := r.gauges.Load(name); ok {
		return v.(otelmetric.Float64Gauge), nil
	}
	var opts []otelmetric.Float64GaugeOption
	if unit != "" {
		opts = append(opts, otelmetric.WithUnit(unit))
	}
	g, err := r.meter.Float64Gauge(name, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gauge %q: %w", name, err)
	}
	r.gauges.Store(name, g)
	return g, nil
}

func (r *metricsRegistry) getOrCreateHistogram(name, unit string) (otelmetric.Float64Histogram, error) {
	if v, ok := r.histograms.Load(name); ok {
		return v.(otelmetric.Float64Histogram), nil
	}
	var opts []otelmetric.Float64HistogramOption
	if unit != "" {
		opts = append(opts, otelmetric.WithUnit(unit))
	}
	h, err := r.meter.Float64Histogram(name, opts...)
	if err != nil {
		return nil, fmt.Errorf("create histogram %q: %w", name, err)
	}
	r.histograms.Store(name, h)
	return h, nil
}

func (r *metricsRegistry) getOrCreateRate(name string) (otelmetric.Int64Counter, otelmetric.Int64Counter, error) {
	type ratePair struct{ nonZero, total otelmetric.Int64Counter }
	if v, ok := r.rates.Load(name); ok {
		rp := v.(ratePair)
		return rp.nonZero, rp.total, nil
	}
	nz, err := r.meter.Int64Counter(name + ".occurred")
	if err != nil {
		return nil, nil, err
	}
	tot, err := r.meter.Int64Counter(name + ".total")
	if err != nil {
		return nil, nil, err
	}
	r.rates.Store(name, ratePair{nz, tot})
	return nz, tot, nil
}

func tagsToAttributes(t *metrics.TagSet) attribute.Set {
	n := (*atlas.Node)(t)
	if n.Len() < 1 {
		return *attribute.EmptySet()
	}
	kvs := make([]attribute.KeyValue, 0, n.Len())
	for !n.IsRoot() {
		prev, key, value := n.Data()
		n = prev
		if key != "" && value != "" {
			kvs = append(kvs, attribute.String(key, value))
		}
	}
	return attribute.NewSet(kvs...)
}

func unitFor(vt metrics.ValueType) string {
	switch vt {
	case metrics.Time:
		return "ms"
	case metrics.Data:
		return "By"
	default:
		return ""
	}
}

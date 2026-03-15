// Package otel implements a k6 output extension that exports metrics and traces
// to any OTLP-compatible backend via gRPC or HTTP. It also injects W3C Baggage
// (k6.test.name, k6.test.step, k6.test.vu, k6.test.iteration) into outgoing
// HTTP requests so downstream services can correlate load test traffic.
package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.k6.io/k6/output"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Output implements go.k6.io/k6/output.Output and output.WithStopWithTestError.
type Output struct {
	output.SampleBuffer

	config Config
	logger logrus.FieldLogger

	meterProvider *sdkmetric.MeterProvider
	traceProvider *sdktrace.TracerProvider

	metricsReg *metricsRegistry
	tracingMgr *tracingManager
	flusher    *output.PeriodicFlusher
}

var _ output.WithStopWithTestError = (*Output)(nil)

// New creates a new Output from k6 output.Params.
func New(p output.Params) (*Output, error) {
	cfg, err := NewConfigFromEnv(p.Environment, p.JSONConfig)
	if err != nil {
		return nil, err
	}

	return &Output{
		config: cfg,
		logger: p.Logger,
	}, nil
}

// Description returns a human-readable description shown in `k6 run`.
func (o *Output) Description() string {
	return fmt.Sprintf("opentelemetry (%s)", o.config)
}

// Start initializes the OTel providers and starts the periodic flusher.
func (o *Output) Start() error {
	o.logger.Info("Starting OpenTelemetry output...")

	ctx := context.Background()

	res, err := resource.Merge(resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName(o.config.ServiceName),
			semconv.ServiceVersion(o.config.ServiceVersion),
		))
	if err != nil {
		return fmt.Errorf("create OTel resource: %w", err)
	}

	// Metrics provider
	if o.config.MetricsEnabled {
		metricExp, err := newMetricExporter(ctx, o.config)
		if err != nil {
			return fmt.Errorf("create metric exporter: %w", err)
		}

		o.meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp,
				sdkmetric.WithInterval(o.config.ExportInterval),
			)),
		)

		o.metricsReg = newMetricsRegistry(
			o.meterProvider.Meter("k6"),
			o.config.MetricPrefix,
		)
	}

	// Trace provider
	if o.config.TracesEnabled {
		traceExp, err := newTraceExporter(ctx, o.config)
		if err != nil {
			return fmt.Errorf("create trace exporter: %w", err)
		}

		sampler := sdktrace.AlwaysSample()
		if o.config.SampleRate < 1.0 {
			sampler = sdktrace.TraceIDRatioBased(o.config.SampleRate)
		}

		o.traceProvider = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(traceExp),
			sdktrace.WithSampler(sampler),
		)

		otel.SetTracerProvider(o.traceProvider)
		o.tracingMgr = newTracingManager(
			o.traceProvider.Tracer("k6"),
			o.config,
		)
	}

	pf, err := output.NewPeriodicFlusher(o.config.FlushInterval, o.flush)
	if err != nil {
		return err
	}
	o.flusher = pf

	o.logger.Info("OpenTelemetry output started")
	return nil
}

// StopWithTestError flushes remaining data and shuts down providers.
func (o *Output) StopWithTestError(_ error) error {
	o.logger.Info("Stopping OpenTelemetry output...")

	o.flusher.Stop()

	if o.tracingMgr != nil {
		o.tracingMgr.endAllSpans()
	}

	ctx := context.Background()

	if o.traceProvider != nil {
		if err := o.traceProvider.Shutdown(ctx); err != nil {
			o.logger.WithError(err).Error("trace provider shutdown error")
		}
	}

	if o.meterProvider != nil {
		if err := o.meterProvider.Shutdown(ctx); err != nil {
			o.logger.WithError(err).Error("meter provider shutdown error")
		}
	}

	o.logger.Info("OpenTelemetry output stopped")
	return nil
}

// Stop implements output.Output (delegates to StopWithTestError).
func (o *Output) Stop() error {
	return o.StopWithTestError(nil)
}

// flush reads buffered samples and dispatches them to metrics and tracing.
func (o *Output) flush() {
	samples := o.GetBufferedSamples()
	start := time.Now()
	var count int

	ctx := context.Background()

	for _, sc := range samples {
		for _, sample := range sc.GetSamples() {
			count++

			if o.metricsReg != nil {
				if err := o.metricsReg.record(ctx, sample); err != nil {
					o.logger.WithError(err).Warn("metric dispatch error")
				}
			}

			if o.tracingMgr != nil {
				o.tracingMgr.recordSample(ctx, sample)
			}
		}
	}

	if count > 0 {
		o.logger.WithField("t", time.Since(start)).
			WithField("samples", count).
			Debug("flushed samples")
	}
}

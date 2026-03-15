package otel

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/mstoykov/atlas"
	"go.k6.io/k6/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// tracingManager creates OTel spans from k6 metric samples.
type tracingManager struct {
	tracer  trace.Tracer
	cfg     Config
	mu      sync.Mutex
	vuSpans map[string]spanEntry // key: "vu:iteration"
}

type spanEntry struct {
	span trace.Span
	ctx  context.Context
}

func newTracingManager(tracer trace.Tracer, cfg Config) *tracingManager {
	return &tracingManager{
		tracer:  tracer,
		cfg:     cfg,
		vuSpans: make(map[string]spanEntry),
	}
}

// recordSample creates spans from k6 metric samples.
func (tm *tracingManager) recordSample(ctx context.Context, sample metrics.Sample) {
	tags := extractTags(sample.Tags)

	switch sample.Metric.Name {
	case "http_reqs":
		tm.recordHTTPSpan(ctx, tags, sample)
	case "http_req_failed":
		if sample.Value != 0 {
			tm.recordErrorSpan(ctx, tags, sample)
		}
	case "checks":
		tm.recordCheckSpan(ctx, tags, sample)
	case "iterations":
		tm.endIterationSpan(tags)
	}
}

func (tm *tracingManager) recordHTTPSpan(ctx context.Context, tags map[string]string, sample metrics.Sample) {
	method := tags["method"]
	status := tags["status"]
	name := tags["name"]
	group := tagOrDefault(tags, "group", "default")

	spanName := fmt.Sprintf("HTTP %s %s", method, name)
	parentCtx := tm.getOrCreateIterationCtx(ctx, tags)

	_, span := tm.tracer.Start(parentCtx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", tags["url"]),
			attribute.String("http.status_code", status),
			attribute.String("k6.test.step", group),
			attribute.String("k6.test.name", tm.cfg.ServiceName),
			attribute.Float64("http.response_time_ms", sample.Value),
		),
	)

	statusCode, _ := strconv.Atoi(status)
	if statusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
	}
	span.End()
}

func (tm *tracingManager) recordCheckSpan(ctx context.Context, tags map[string]string, sample metrics.Sample) {
	checkName := tags["check"]
	group := tagOrDefault(tags, "group", "default")
	parentCtx := tm.getOrCreateIterationCtx(ctx, tags)

	_, span := tm.tracer.Start(parentCtx, "check: "+checkName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("k6.check.name", checkName),
			attribute.String("k6.test.step", group),
			attribute.Bool("k6.check.passed", sample.Value != 0),
		),
	)
	if sample.Value == 0 {
		span.SetStatus(codes.Error, "check failed: "+checkName)
	}
	span.End()
}

func (tm *tracingManager) recordErrorSpan(ctx context.Context, tags map[string]string, _ metrics.Sample) {
	method := tags["method"]
	name := tags["name"]
	errMsg := tags["error"]
	if errMsg == "" {
		errMsg = tags["error_code"]
	}
	parentCtx := tm.getOrCreateIterationCtx(ctx, tags)

	_, span := tm.tracer.Start(parentCtx, fmt.Sprintf("ERROR %s %s", method, name),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", tags["url"]),
			attribute.String("k6.error", errMsg),
		),
	)
	span.SetStatus(codes.Error, errMsg)
	span.End()
}

// getOrCreateIterationCtx returns a context with a parent span for the VU iteration.
func (tm *tracingManager) getOrCreateIterationCtx(ctx context.Context, tags map[string]string) context.Context {
	vu := tags["vu"]
	iter := tags["iter"]
	key := vu + ":" + iter

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if entry, ok := tm.vuSpans[key]; ok {
		return entry.ctx
	}

	group := tagOrDefault(tags, "group", "default")
	scenario := tags["scenario"]

	newCtx, span := tm.tracer.Start(ctx, "k6.iteration",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("k6.test.name", tm.cfg.ServiceName),
			attribute.String("k6.test.vu", vu),
			attribute.String("k6.test.iteration", iter),
			attribute.String("k6.test.step", group),
			attribute.String("k6.scenario", scenario),
		),
	)

	tm.vuSpans[key] = spanEntry{span: span, ctx: newCtx}
	return newCtx
}

func (tm *tracingManager) endIterationSpan(tags map[string]string) {
	key := tags["vu"] + ":" + tags["iter"]
	tm.mu.Lock()
	entry, ok := tm.vuSpans[key]
	if ok {
		delete(tm.vuSpans, key)
	}
	tm.mu.Unlock()

	if ok {
		entry.span.End()
	}
}

func (tm *tracingManager) endAllSpans() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	for k, entry := range tm.vuSpans {
		entry.span.End()
		delete(tm.vuSpans, k)
	}
}

// extractTags reads all k/v pairs from a k6 TagSet (which is *atlas.Node).
func extractTags(t *metrics.TagSet) map[string]string {
	tags := make(map[string]string)
	n := (*atlas.Node)(t)
	for !n.IsRoot() {
		prev, key, value := n.Data()
		n = prev
		if key != "" {
			tags[key] = value
		}
	}
	return tags
}

func tagOrDefault(tags map[string]string, key, def string) string {
	if v, ok := tags[key]; ok && v != "" {
		return v
	}
	return def
}

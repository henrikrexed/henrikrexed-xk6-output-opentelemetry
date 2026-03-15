# Configuration Reference

All configuration is via environment variables with the `K6_OTEL_` prefix.

## Endpoint

| Variable | Default | Description |
|----------|---------|-------------|
| `K6_OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTLP endpoint (host:port, no scheme) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | | Standard OTel env var (also supported) |
| `K6_OTEL_EXPORTER_TYPE` | `grpc` | Protocol: `grpc` or `http` |
| `K6_OTEL_GRPC_EXPORTER_INSECURE` | `false` | Disable TLS for gRPC |
| `K6_OTEL_HTTP_EXPORTER_INSECURE` | `false` | Disable TLS for HTTP |
| `K6_OTEL_HEADERS` | | Headers: `k1=v1,k2=v2` (URL-encoded) |

## Service Identity

| Variable | Default | Description |
|----------|---------|-------------|
| `K6_OTEL_SERVICE_NAME` | `k6` | OTel service.name resource attribute |
| `OTEL_SERVICE_NAME` | | Standard OTel env var (also supported) |
| `K6_OTEL_SERVICE_VERSION` | `1.6.1` | OTel service.version |

## Feature Toggles

| Variable | Default | Description |
|----------|---------|-------------|
| `K6_OTEL_TRACES_ENABLED` | `true` | Export traces/spans |
| `K6_OTEL_METRICS_ENABLED` | `true` | Export metrics |
| `K6_OTEL_BAGGAGE_ENABLED` | `true` | Inject W3C Baggage headers |
| `K6_OTEL_TRACES_SAMPLE_RATE` | `1.0` | Trace sampling rate (0.0 to 1.0) |

## Intervals

| Variable | Default | Description |
|----------|---------|-------------|
| `K6_OTEL_FLUSH_INTERVAL` | `1s` | How often k6 flushes its sample buffer |
| `K6_OTEL_EXPORT_INTERVAL` | `10s` | How often the OTel SDK exports metrics |

## Metrics

| Variable | Default | Description |
|----------|---------|-------------|
| `K6_OTEL_METRIC_PREFIX` | `` | Prefix added to all metric names |

## Examples

### Metrics only (no traces)

```bash
K6_OTEL_TRACES_ENABLED=false \
K6_OTEL_GRPC_EXPORTER_INSECURE=true \
./k6 run --out opentelemetry test.js
```

### Traces only (no metrics)

```bash
K6_OTEL_METRICS_ENABLED=false \
K6_OTEL_GRPC_EXPORTER_INSECURE=true \
./k6 run --out opentelemetry test.js
```

### 10% trace sampling

```bash
K6_OTEL_TRACES_SAMPLE_RATE=0.1 \
./k6 run --out opentelemetry test.js
```

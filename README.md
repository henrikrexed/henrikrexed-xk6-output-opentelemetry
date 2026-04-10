# xk6-output-opentelemetry

📖 **[Full Documentation](https://henrikrexed.github.io/henrikrexed-xk6-output-opentelemetry/)** — Getting started, configuration, JS API reference, examples

A [k6](https://k6.io/) output extension that exports **both metrics AND traces** to any OTLP-compatible backend via gRPC or HTTP, with W3C Baggage injection for downstream service correlation.

## Features

- **Metrics**: All k6 built-in metrics exported as OTel metrics (counters, gauges, histograms)
- **Traces**: Per-iteration parent spans, per-HTTP-request child spans, per-check spans
- **W3C Baggage**: Injects `k6.test.name`, `k6.test.step`, `k6.test.vu`, `k6.test.iteration` on outgoing requests
- **W3C TraceContext**: Propagates trace context so downstream services link to k6 traces
- **Dual protocol**: gRPC and HTTP OTLP export
- **Configurable**: Feature toggles for traces, metrics, and baggage independently

## Install

### Pre-built binaries

Download a ready-to-use k6 binary (with the extension included) from the
[Releases](https://github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry/releases) page.
Binaries are available for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64).

```bash
curl -LO https://github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry/releases/latest/download/k6-linux-amd64
chmod +x k6-linux-amd64 && mv k6-linux-amd64 k6
```

### Docker

```bash
docker pull ghcr.io/henrikrexed/henrikrexed-xk6-output-opentelemetry:latest

docker run --rm \
  -e K6_OTEL_GRPC_EXPORTER_INSECURE=true \
  -e K6_OTEL_EXPORTER_OTLP_ENDPOINT=host.docker.internal:4317 \
  -v $(pwd)/test.js:/home/k6/test.js \
  ghcr.io/henrikrexed/henrikrexed-xk6-output-opentelemetry:latest \
  run --out opentelemetry /home/k6/test.js
```

### Build with xk6

Combine with other k6 extensions using [xk6](https://github.com/grafana/xk6):

```bash
xk6 build latest --with github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry
```

### Build from source

```bash
git clone https://github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry.git
cd henrikrexed-xk6-output-opentelemetry
make build
```

## Usage

```bash
K6_OTEL_GRPC_EXPORTER_INSECURE=true \
K6_OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
./k6 run --out opentelemetry test.js
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `K6_OTEL_SERVICE_NAME` | `k6` | Service name in OTel resource |
| `K6_OTEL_SERVICE_VERSION` | `1.6.1` | Service version |
| `K6_OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTLP endpoint (host:port) |
| `K6_OTEL_EXPORTER_TYPE` | `grpc` | Protocol: `grpc` or `http` |
| `K6_OTEL_GRPC_EXPORTER_INSECURE` | `false` | Use insecure gRPC |
| `K6_OTEL_HTTP_EXPORTER_INSECURE` | `false` | Use insecure HTTP |
| `K6_OTEL_HEADERS` | | Headers in `k1=v1,k2=v2` format |
| `K6_OTEL_TRACES_ENABLED` | `true` | Export traces/spans |
| `K6_OTEL_METRICS_ENABLED` | `true` | Export metrics |
| `K6_OTEL_BAGGAGE_ENABLED` | `true` | Inject W3C Baggage headers |
| `K6_OTEL_TRACES_SAMPLE_RATE` | `1.0` | Trace sampling rate (0.0-1.0) |
| `K6_OTEL_METRIC_PREFIX` | | Prefix for metric names |
| `K6_OTEL_FLUSH_INTERVAL` | `1s` | k6 sample buffer flush interval |
| `K6_OTEL_EXPORT_INTERVAL` | `10s` | OTel metric export interval |

Also supports `OTEL_SERVICE_NAME` and `OTEL_EXPORTER_OTLP_ENDPOINT` (standard OTel env vars).

## Trace Structure

```
k6.iteration (parent span per VU iteration)
├── HTTP GET /api/products          (per-request span)
├── check: status is 200            (per-check span)
├── HTTP GET /api/product/ABC123
├── HTTP POST /api/cart/add
└── HTTP POST /api/checkout
```

### Span Attributes

| Attribute | Description |
|-----------|-------------|
| `k6.test.name` | Service/test name |
| `k6.test.vu` | Virtual user ID |
| `k6.test.iteration` | Iteration number |
| `k6.test.step` | k6 group name or "default" |
| `k6.scenario` | k6 scenario name |
| `http.method` | HTTP method |
| `http.url` | Request URL |
| `http.status_code` | Response status code |
| `k6.check.name` | Check name (for check spans) |
| `k6.check.passed` | Whether check passed |

## W3C Baggage

When `K6_OTEL_BAGGAGE_ENABLED=true`, the following baggage entries are injected on every outgoing HTTP request via the `baggage` header:

- `k6.test.name` — test/service name
- `k6.test.step` — current k6 group or "default"
- `k6.test.vu` — VU ID
- `k6.test.iteration` — iteration number

Downstream services that read W3C Baggage can use these to identify load test traffic.

## License

Apache License 2.0

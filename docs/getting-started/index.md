# Getting Started

## Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [xk6](https://github.com/grafana/xk6) (`go install go.k6.io/xk6/cmd/xk6@latest`)
- An OTLP-compatible backend (Jaeger, Grafana Cloud, etc.) or the OTel Collector

## Build

```bash
xk6 build v1.6.1 --with github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry
```

This produces a `k6` binary with the extension built in.

### Build from source

```bash
git clone https://github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry.git
cd xk6-output-opentelemetry
make build
```

## Run

```bash
# Start a local collector (e.g., with Jaeger)
docker run -d --name jaeger \
  -p 16686:16686 -p 4317:4317 \
  jaegertracing/all-in-one:1.56

# Run k6 with OTel output
K6_OTEL_GRPC_EXPORTER_INSECURE=true \
K6_OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
./k6 run --out opentelemetry test.js
```

Open [http://localhost:16686](http://localhost:16686) to see traces in Jaeger.

## Verify

The k6 output line should show:

```
opentelemetry (grpc localhost:4317 [traces+metrics+baggage])
```

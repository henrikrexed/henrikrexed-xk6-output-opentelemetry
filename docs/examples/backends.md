# Backend Setup

## Jaeger (Local)

```bash
docker run -d --name jaeger \
  -p 16686:16686 -p 4317:4317 \
  jaegertracing/all-in-one:1.56

K6_OTEL_GRPC_EXPORTER_INSECURE=true \
K6_OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
./k6 run --out opentelemetry test.js
```

Open [http://localhost:16686](http://localhost:16686) — search for service `k6`.

## Grafana Cloud

```bash
K6_OTEL_EXPORTER_TYPE=http \
K6_OTEL_EXPORTER_OTLP_ENDPOINT=otlp-gateway-prod-us-central-0.grafana.net \
K6_OTEL_HEADERS="Authorization=Basic <base64-instanceId:apiKey>" \
./k6 run --out opentelemetry test.js
```

## Dynatrace

```bash
K6_OTEL_EXPORTER_TYPE=http \
K6_OTEL_EXPORTER_OTLP_ENDPOINT=<env-id>.live.dynatrace.com/api/v2/otlp \
K6_OTEL_HEADERS="Authorization=Api-Token <token>" \
./k6 run --out opentelemetry test.js
```

## OTel Collector

Point k6 at your collector:

```bash
K6_OTEL_GRPC_EXPORTER_INSECURE=true \
K6_OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317 \
./k6 run --out opentelemetry test.js
```

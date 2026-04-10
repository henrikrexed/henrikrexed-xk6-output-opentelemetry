# Getting Started

## Prerequisites

- An OTLP-compatible backend (Jaeger, Grafana Cloud, etc.) or the OTel Collector

## Installation

There are several ways to get a k6 binary with the OpenTelemetry output extension.

### Option 1: Download a pre-built binary

Pre-built binaries are available for Linux, macOS, and Windows from the
[GitHub Releases](https://github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry/releases) page.

| Platform | Architecture | Asset |
|----------|-------------|-------|
| Linux | amd64 | `k6-linux-amd64` |
| Linux | arm64 | `k6-linux-arm64` |
| macOS | arm64 (Apple Silicon) | `k6-darwin-arm64` |
| macOS | amd64 (Intel) | `k6-darwin-amd64` |
| Windows | amd64 | `k6-windows-amd64.exe` |

```bash
# Example: download the latest Linux amd64 binary
curl -LO https://github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry/releases/latest/download/k6-linux-amd64
chmod +x k6-linux-amd64
mv k6-linux-amd64 k6
```

### Option 2: Use the Docker image

A multi-arch Docker image (linux/amd64 and linux/arm64) is published to GitHub Container Registry on every release.

```bash
# Pull the latest image
docker pull ghcr.io/henrikrexed/henrikrexed-xk6-output-opentelemetry:latest

# Run a test
docker run --rm \
  -e K6_OTEL_GRPC_EXPORTER_INSECURE=true \
  -e K6_OTEL_EXPORTER_OTLP_ENDPOINT=host.docker.internal:4317 \
  -v $(pwd)/test.js:/home/k6/test.js \
  ghcr.io/henrikrexed/henrikrexed-xk6-output-opentelemetry:latest \
  run --out opentelemetry /home/k6/test.js
```

You can also pin to a specific version tag (e.g., `ghcr.io/henrikrexed/henrikrexed-xk6-output-opentelemetry:0.0.1`).

### Option 3: Build with xk6

If you need to combine this extension with other k6 extensions, use [xk6](https://github.com/grafana/xk6):

**Prerequisites:** [Go 1.24+](https://go.dev/dl/), [xk6](https://github.com/grafana/xk6) (`go install go.k6.io/xk6/cmd/xk6@latest`)

```bash
xk6 build latest --with github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry
```

This produces a `k6` binary with the extension built in.

### Option 4: Build from source

```bash
git clone https://github.com/henrikrexed/henrikrexed-xk6-output-opentelemetry.git
cd henrikrexed-xk6-output-opentelemetry
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

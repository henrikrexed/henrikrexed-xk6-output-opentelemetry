# Architecture

## Data Flow

```mermaid
flowchart TB
    subgraph k6["k6 Runtime"]
        Script["Test Script (JS)"]
        Engine["k6 Engine"]
        SB["Sample Buffer"]
        JSMod["k6/x/otel Module"]
    end

    subgraph Extension["xk6-output-opentelemetry"]
        Flusher["Periodic Flusher"]
        MetricsReg["Metrics Registry"]
        TracingMgr["Tracing Manager"]
        MeterProvider["OTel MeterProvider"]
        TraceProvider["OTel TracerProvider"]
    end

    subgraph Export["OTLP Export"]
        MetricExp["Metric Exporter<br/>(gRPC / HTTP)"]
        TraceExp["Trace Exporter<br/>(gRPC / HTTP)"]
    end

    Script --> Engine
    Engine --> SB
    JSMod -->|"setBaggage / setAttribute"| TracingMgr
    SB --> Flusher
    Flusher --> MetricsReg
    Flusher --> TracingMgr
    MetricsReg --> MeterProvider --> MetricExp
    TracingMgr --> TraceProvider --> TraceExp

    MetricExp --> Collector["OTel Collector / Backend"]
    TraceExp --> Collector
```

## Trace Structure

```mermaid
gantt
    title k6 Trace Structure (single iteration)
    dateFormat X
    axisFormat %s

    section Lifecycle
    k6.iteration               :0, 100

    section HTTP Requests
    HTTP GET /api/products     :5, 20
    HTTP GET /api/product/ABC  :25, 40
    HTTP POST /api/cart/add    :45, 60
    HTTP POST /api/checkout    :65, 95

    section Checks
    check: status is 200       :21, 23
    check: has items           :41, 43
```

## Metrics Pipeline

```mermaid
flowchart LR
    subgraph k6 Metrics
        Counter["k6 Counter<br/>(http_reqs, iterations)"]
        Gauge["k6 Gauge<br/>(vus, vus_max)"]
        Trend["k6 Trend<br/>(http_req_duration)"]
        Rate["k6 Rate<br/>(http_req_failed, checks)"]
    end

    subgraph OTel Metrics
        OCounter["Float64Counter"]
        OGauge["Float64Gauge"]
        OHist["Float64Histogram"]
        ORateNZ["Int64Counter<br/>(.occurred)"]
        ORateT["Int64Counter<br/>(.total)"]
    end

    Counter --> OCounter
    Gauge --> OGauge
    Trend --> OHist
    Rate --> ORateNZ
    Rate --> ORateT
```

## Baggage Flow

```mermaid
sequenceDiagram
    participant Script as k6 Script
    participant Ext as Extension
    participant App as Application
    participant Backend as OTel Backend

    Script->>Ext: otel.setBaggage("user.type", "premium")
    Script->>App: HTTP GET /api/products
    Note over Ext: Injects headers:<br/>traceparent: 00-trace...<br/>baggage: k6.test.name=k6,<br/>k6.test.vu=1,user.type=premium
    Ext->>App: HTTP request with headers
    App->>Backend: Trace span (reads baggage)
    Ext->>Backend: k6 trace span + metrics
```

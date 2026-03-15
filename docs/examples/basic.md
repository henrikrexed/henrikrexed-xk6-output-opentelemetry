# Basic Usage

## Simple test with OTel output

```javascript
import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: 5,
  duration: "30s",
};

export default function () {
  const res = http.get("http://localhost:8080/api/products");
  check(res, {
    "status is 200": (r) => r.status === 200,
  });
  sleep(1);
}
```

Run:

```bash
K6_OTEL_GRPC_EXPORTER_INSECURE=true \
K6_OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
./k6 run --out opentelemetry test.js
```

This produces:

- **Metrics**: `http_reqs`, `http_req_duration`, `http_req_failed`, `checks`, `vus`, etc.
- **Traces**: One `k6.iteration` span per iteration, with child `HTTP GET /api/products` and `check: status is 200` spans
- **Baggage**: `k6.test.name=k6`, `k6.test.vu=<id>`, `k6.test.iteration=<n>` on each HTTP request

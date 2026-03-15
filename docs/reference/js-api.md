# JavaScript API Reference

The extension registers a k6 module at `k6/x/otel` that exposes functions for setting custom OTel baggage and span attributes from your test scripts.

## Import

```javascript
import otel from "k6/x/otel";
```

## Functions

### `otel.setBaggage(key, value)`

Sets a custom W3C Baggage entry. All subsequent HTTP requests made by this VU will include these baggage entries in the `baggage` HTTP header.

```javascript
otel.setBaggage("user.type", "premium");
otel.setBaggage("test.environment", "staging");
```

### `otel.setAttribute(key, value)`

Sets a custom span attribute that will be added to the current iteration span.

```javascript
otel.setAttribute("business.flow", "checkout");
otel.setAttribute("test.variant", "A");
```

## Built-in Baggage

When `K6_OTEL_BAGGAGE_ENABLED=true` (default), these baggage entries are automatically injected:

| Key | Value | Description |
|-----|-------|-------------|
| `k6.test.name` | Service name | From `K6_OTEL_SERVICE_NAME` |
| `k6.test.step` | Group name | Current k6 `group()` or "default" |
| `k6.test.vu` | VU ID | The virtual user number |
| `k6.test.iteration` | Iteration | Current iteration number |

Custom baggage set via `otel.setBaggage()` is merged with the built-in entries.

## Example

```javascript
import http from "k6/http";
import otel from "k6/x/otel";
import { group } from "k6";

export default function () {
  otel.setBaggage("test.scenario", "browse-and-buy");
  otel.setAttribute("business.flow", "e2e");

  group("browse", function () {
    // baggage header will include: k6.test.step=browse, test.scenario=browse-and-buy
    http.get("http://frontend:8080/api/products");
  });

  group("purchase", function () {
    otel.setBaggage("cart.has_items", "true");
    http.post("http://frontend:8080/api/checkout", JSON.stringify({...}));
  });
}
```

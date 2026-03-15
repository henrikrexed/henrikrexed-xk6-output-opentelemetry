# JavaScript API Reference

The extension registers a k6 module at `k6/x/otel` with both low-level primitives and high-level helpers.

```javascript
import otel from "k6/x/otel";
```

## High-Level Helpers

These compose k6's `group()`, `http.*()`, and `check()` with automatic span/baggage management.

### `otel.step(name, fn)`

Wraps a k6 `group()`: sets `k6.test.step` baggage to the step name, executes the callback, resets step to "default".

```javascript
otel.step("Browse Products", function () {
  // k6.test.step baggage = "Browse Products" for all requests in here
  http.get("http://frontend:8080/api/products");
});
```

### `otel.request(name, method, url, [body], [params])`

Makes an HTTP request with automatic baggage injection. Sets `k6.request.name` baggage. Returns the k6 response.

```javascript
// GET
let res = otel.request("list-products", "GET", "http://frontend:8080/api/products");

// POST with body and headers
otel.request("add-to-cart", "POST", "http://frontend:8080/api/cart/add",
  JSON.stringify({ productId: "ABC", quantity: 1 }),
  { headers: { "Content-Type": "application/json" } }
);
```

### `otel.check(name, response, checks)`

Wraps k6 `check()` with a named check group. Sets `k6.check.group` attribute.

```javascript
let res = otel.request("list-products", "GET", url);
otel.check("products-ok", res, {
  "status is 200": (r) => r.status === 200,
  "has products": (r) => r.json().products.length > 0,
});
```

## Low-Level Primitives

### `otel.setBaggage(key, value)`

Sets a custom W3C Baggage entry injected on all subsequent HTTP requests.

```javascript
otel.setBaggage("user.type", "premium");
otel.setBaggage("test.environment", "staging");
```

### `otel.setAttribute(key, value)`

Sets a custom span attribute on the current iteration span.

```javascript
otel.setAttribute("business.flow", "checkout");
```

## Built-in Baggage

When `K6_OTEL_BAGGAGE_ENABLED=true` (default), these are automatically injected:

| Key | Value | Description |
|-----|-------|-------------|
| `k6.test.name` | Service name | From `K6_OTEL_SERVICE_NAME` |
| `k6.test.step` | Step name | Set by `otel.step()` or "default" |
| `k6.test.vu` | VU ID | Virtual user number |
| `k6.test.iteration` | Iteration | Current iteration number |

## Before / After Comparison

=== "Before (raw k6)"

    ```javascript
    import http from "k6/http";
    import { check, group, sleep } from "k6";

    export default function () {
      group("Browse Products", function () {
        const res = http.get("http://frontend:8080/api/products");
        check(res, { "status is 200": (r) => r.status === 200 });
      });
      sleep(1);

      group("Add to Cart", function () {
        http.post("http://frontend:8080/api/cart/add",
          JSON.stringify({ productId: "ABC", quantity: 1 }),
          { headers: { "Content-Type": "application/json" } }
        );
      });
    }
    ```

=== "After (otel helpers)"

    ```javascript
    import otel from "k6/x/otel";
    import { sleep } from "k6";

    export default function () {
      otel.setBaggage("test.scenario", "e2e");

      otel.step("Browse Products", () => {
        let res = otel.request("list-products", "GET", "http://frontend:8080/api/products");
        otel.check("products-ok", res, { "status is 200": (r) => r.status === 200 });
      });
      sleep(1);

      otel.step("Add to Cart", () => {
        otel.request("add-to-cart", "POST", "http://frontend:8080/api/cart/add",
          JSON.stringify({ productId: "ABC", quantity: 1 }),
          { headers: { "Content-Type": "application/json" } }
        );
      });
    }
    ```

The helper version automatically sets baggage, names spans, and groups checks — with zero extra boilerplate.

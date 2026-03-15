# Custom Baggage

Use the `k6/x/otel` JS module to inject custom W3C Baggage entries that downstream services can read.

```javascript
import http from "k6/http";
import otel from "k6/x/otel";
import { group, check } from "k6";

export default function () {
  // Set custom baggage — injected on all subsequent HTTP requests
  otel.setBaggage("test.scenario", "browse-and-buy");
  otel.setBaggage("test.environment", "staging");

  // Set custom span attribute — added to the iteration span
  otel.setAttribute("business.flow", "e2e-checkout");

  group("browse", function () {
    // baggage header: k6.test.step=browse, test.scenario=browse-and-buy, ...
    const res = http.get("http://frontend:8080/api/products");
    check(res, { "products loaded": (r) => r.status === 200 });
  });

  group("purchase", function () {
    otel.setBaggage("cart.has_items", "true");

    http.post(
      "http://frontend:8080/api/checkout",
      JSON.stringify({
        email: "test@example.com",
        creditCardNumber: "4111111111111111",
        creditCardCvv: "123",
        creditCardExpirationYear: "2030",
        creditCardExpirationMonth: "12",
      }),
      { headers: { "Content-Type": "application/json" } }
    );
  });
}
```

## Reading Baggage in Downstream Services

Any service that reads W3C Baggage headers will see:

```
baggage: k6.test.name=k6,k6.test.vu=1,k6.test.iteration=0,k6.test.step=browse,test.scenario=browse-and-buy,test.environment=staging
```

In the OpenTelemetry Demo Light, the Product Catalog service (Go) reads incoming baggage and attaches it as span attributes — so you'll see `test.scenario=browse-and-buy` on the product catalog trace spans.

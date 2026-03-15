// Package xk6otel registers the xk6-output-opentelemetry extension with k6.
package xk6otel

import (
	"github.com/henrikrexed/xk6-output-opentelemetry/pkg/otel"
	"go.k6.io/k6/output"

	// Import the JS module so its init() runs and registers "k6/x/otel"
	_ "github.com/henrikrexed/xk6-output-opentelemetry/pkg/jsmodule"
)

func init() {
	output.RegisterExtension("opentelemetry", func(p output.Params) (output.Output, error) {
		return otel.New(p)
	})
}

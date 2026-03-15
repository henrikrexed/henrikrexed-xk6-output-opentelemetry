package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "k6", cfg.ServiceName)
	assert.Equal(t, "1.6.1", cfg.ServiceVersion)
	assert.Equal(t, "grpc", cfg.Protocol)
	assert.Equal(t, "localhost:4317", cfg.Endpoint)
	assert.True(t, cfg.TracesEnabled)
	assert.True(t, cfg.MetricsEnabled)
	assert.True(t, cfg.BaggageEnabled)
	assert.Equal(t, 1.0, cfg.SampleRate)
}

func TestNewConfigFromEnv_Defaults(t *testing.T) {
	cfg, err := NewConfigFromEnv(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "k6", cfg.ServiceName)
	assert.True(t, cfg.TracesEnabled)
}

func TestNewConfigFromEnv_Overrides(t *testing.T) {
	env := map[string]string{
		"K6_OTEL_SERVICE_NAME":             "my-test",
		"K6_OTEL_EXPORTER_OTLP_ENDPOINT":  "collector:4317",
		"K6_OTEL_GRPC_EXPORTER_INSECURE":  "true",
		"K6_OTEL_TRACES_ENABLED":          "false",
		"K6_OTEL_METRICS_ENABLED":         "true",
		"K6_OTEL_BAGGAGE_ENABLED":         "false",
		"K6_OTEL_TRACES_SAMPLE_RATE":      "0.5",
		"K6_OTEL_METRIC_PREFIX":           "k6.",
	}

	cfg, err := NewConfigFromEnv(env, nil)
	require.NoError(t, err)
	assert.Equal(t, "my-test", cfg.ServiceName)
	assert.Equal(t, "collector:4317", cfg.Endpoint)
	assert.True(t, cfg.Insecure)
	assert.False(t, cfg.TracesEnabled)
	assert.True(t, cfg.MetricsEnabled)
	assert.False(t, cfg.BaggageEnabled)
	assert.Equal(t, 0.5, cfg.SampleRate)
	assert.Equal(t, "k6.", cfg.MetricPrefix)
}

func TestNewConfigFromEnv_OTELServiceName(t *testing.T) {
	env := map[string]string{
		"OTEL_SERVICE_NAME": "otel-svc",
	}
	cfg, err := NewConfigFromEnv(env, nil)
	require.NoError(t, err)
	assert.Equal(t, "otel-svc", cfg.ServiceName)
}

func TestNewConfigFromEnv_StripScheme(t *testing.T) {
	env := map[string]string{
		"K6_OTEL_EXPORTER_OTLP_ENDPOINT": "http://collector:4317",
	}
	cfg, err := NewConfigFromEnv(env, nil)
	require.NoError(t, err)
	assert.Equal(t, "collector:4317", cfg.Endpoint)
}

func TestConfig_Validate_BadProtocol(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Protocol = "websocket"
	assert.Error(t, cfg.Validate())
}

func TestConfig_Validate_BadSampleRate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SampleRate = 1.5
	assert.Error(t, cfg.Validate())
}

func TestConfig_Validate_EmptyEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Endpoint = ""
	assert.Error(t, cfg.Validate())
}

func TestConfig_ParseHeaders(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Headers = "Authorization=Bearer%20tok,X-Custom=val"
	h, err := cfg.ParseHeaders()
	require.NoError(t, err)
	assert.Equal(t, "Bearer tok", h["Authorization"])
	assert.Equal(t, "val", h["X-Custom"])
}

func TestConfig_String(t *testing.T) {
	cfg := DefaultConfig()
	s := cfg.String()
	assert.Contains(t, s, "grpc")
	assert.Contains(t, s, "traces")
	assert.Contains(t, s, "metrics")
	assert.Contains(t, s, "baggage")
}

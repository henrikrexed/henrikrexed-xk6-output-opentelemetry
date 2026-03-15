package otel

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	ProtoGRPC = "grpc"
	ProtoHTTP = "http"
)

// Config holds all configuration for the OTel output extension.
type Config struct {
	// Service identity
	ServiceName    string `json:"serviceName"`
	ServiceVersion string `json:"serviceVersion"`

	// Endpoint
	Endpoint string `json:"endpoint"` // host:port (no scheme)
	Protocol string `json:"protocol"` // "grpc" or "http"
	Insecure bool   `json:"insecure"`
	Headers  string `json:"headers"` // k1=v1,k2=v2

	// Feature toggles
	TracesEnabled  bool    `json:"tracesEnabled"`
	MetricsEnabled bool    `json:"metricsEnabled"`
	BaggageEnabled bool    `json:"baggageEnabled"`
	SampleRate     float64 `json:"sampleRate"`

	// Intervals
	FlushInterval  time.Duration `json:"flushInterval"`
	ExportInterval time.Duration `json:"exportInterval"`

	// Metric prefix
	MetricPrefix string `json:"metricPrefix"`

	// TLS (paths)
	TLSCert       string `json:"tlsCert"`
	TLSKey        string `json:"tlsKey"`
	TLSCA         string `json:"tlsCA"`
	TLSSkipVerify bool   `json:"tlsSkipVerify"`
}

// DefaultConfig returns config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ServiceName:    "k6",
		ServiceVersion: "1.6.1",
		Endpoint:       "localhost:4317",
		Protocol:       ProtoGRPC,
		Insecure:       false,
		TracesEnabled:  true,
		MetricsEnabled: true,
		BaggageEnabled: true,
		SampleRate:     1.0,
		FlushInterval:  1 * time.Second,
		ExportInterval: 10 * time.Second,
	}
}

// NewConfigFromEnv reads config from k6 environment variables.
func NewConfigFromEnv(env map[string]string, jsonRaw json.RawMessage) (Config, error) {
	cfg := DefaultConfig()

	// Apply JSON config first
	if jsonRaw != nil {
		var jc Config
		if err := json.Unmarshal(jsonRaw, &jc); err != nil {
			return cfg, fmt.Errorf("parse JSON config: %w", err)
		}
		cfg = cfg.merge(jc)
	}

	// Environment variables override (K6_OTEL_* prefix)
	if v, ok := env["OTEL_SERVICE_NAME"]; ok && v != "" {
		cfg.ServiceName = v
	}
	if v, ok := env["K6_OTEL_SERVICE_NAME"]; ok && v != "" {
		cfg.ServiceName = v
	}
	if v, ok := env["K6_OTEL_SERVICE_VERSION"]; ok && v != "" {
		cfg.ServiceVersion = v
	}
	if v, ok := env["K6_OTEL_EXPORTER_OTLP_ENDPOINT"]; ok && v != "" {
		cfg.Endpoint = stripScheme(v)
	}
	if v, ok := env["OTEL_EXPORTER_OTLP_ENDPOINT"]; ok && v != "" {
		cfg.Endpoint = stripScheme(v)
	}
	if v, ok := env["K6_OTEL_EXPORTER_TYPE"]; ok && v != "" {
		cfg.Protocol = v
	}
	if v, ok := env["K6_OTEL_GRPC_EXPORTER_INSECURE"]; ok {
		cfg.Insecure = v == "true" || v == "1"
	}
	if v, ok := env["K6_OTEL_HTTP_EXPORTER_INSECURE"]; ok {
		cfg.Insecure = v == "true" || v == "1"
	}
	if v, ok := env["K6_OTEL_HEADERS"]; ok && v != "" {
		cfg.Headers = v
	}
	if v, ok := env["K6_OTEL_TRACES_ENABLED"]; ok {
		cfg.TracesEnabled = v == "true" || v == "1"
	}
	if v, ok := env["K6_OTEL_METRICS_ENABLED"]; ok {
		cfg.MetricsEnabled = v == "true" || v == "1"
	}
	if v, ok := env["K6_OTEL_BAGGAGE_ENABLED"]; ok {
		cfg.BaggageEnabled = v == "true" || v == "1"
	}
	if v, ok := env["K6_OTEL_TRACES_SAMPLE_RATE"]; ok && v != "" {
		var rate float64
		if _, err := fmt.Sscanf(v, "%f", &rate); err == nil {
			cfg.SampleRate = rate
		}
	}
	if v, ok := env["K6_OTEL_METRIC_PREFIX"]; ok {
		cfg.MetricPrefix = v
	}
	if v, ok := env["K6_OTEL_FLUSH_INTERVAL"]; ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.FlushInterval = d
		}
	}
	if v, ok := env["K6_OTEL_EXPORT_INTERVAL"]; ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ExportInterval = d
		}
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (cfg Config) merge(other Config) Config {
	if other.ServiceName != "" {
		cfg.ServiceName = other.ServiceName
	}
	if other.ServiceVersion != "" {
		cfg.ServiceVersion = other.ServiceVersion
	}
	if other.Endpoint != "" {
		cfg.Endpoint = other.Endpoint
	}
	if other.Protocol != "" {
		cfg.Protocol = other.Protocol
	}
	if other.MetricPrefix != "" {
		cfg.MetricPrefix = other.MetricPrefix
	}
	return cfg
}

func (cfg Config) Validate() error {
	if cfg.ServiceName == "" {
		return errors.New("service name is required")
	}
	if cfg.Protocol != ProtoGRPC && cfg.Protocol != ProtoHTTP {
		return fmt.Errorf("unsupported protocol %q, use %q or %q", cfg.Protocol, ProtoGRPC, ProtoHTTP)
	}
	if cfg.Endpoint == "" {
		return errors.New("endpoint is required")
	}
	if cfg.SampleRate < 0 || cfg.SampleRate > 1 {
		return fmt.Errorf("sample rate must be between 0.0 and 1.0, got %f", cfg.SampleRate)
	}
	return nil
}

func (cfg Config) String() string {
	features := ""
	if cfg.TracesEnabled {
		features += "traces+"
	}
	if cfg.MetricsEnabled {
		features += "metrics+"
	}
	if cfg.BaggageEnabled {
		features += "baggage+"
	}
	features = strings.TrimRight(features, "+")
	return fmt.Sprintf("%s %s [%s]", cfg.Protocol, cfg.Endpoint, features)
}

// ParseHeaders parses the k1=v1,k2=v2 format into a map.
func (cfg Config) ParseHeaders() (map[string]string, error) {
	if cfg.Headers == "" {
		return nil, nil
	}
	headers := make(map[string]string)
	for _, h := range strings.Split(cfg.Headers, ",") {
		k, v, ok := strings.Cut(h, "=")
		if !ok {
			return nil, fmt.Errorf("invalid header %q", h)
		}
		key, _ := url.PathUnescape(k)
		val, _ := url.PathUnescape(v)
		headers[key] = val
	}
	return headers, nil
}

func stripScheme(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return endpoint
}

// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"sigs.k8s.io/yaml"
)

//go:embed default_config.yaml
var defaultConfigFS embed.FS

// Metric type constants used in configuration and validation.
const (
	metricTypeGauge     = "gauge"
	metricTypeCounter   = "counter"
	metricTypeHistogram = "histogram"
)

// MetricsConfig is the top-level YAML configuration for config-driven metrics.
type MetricsConfig struct {
	Metrics []MetricMapping `json:"metrics"`
}

// MetricMapping defines how a set of Redis keys maps to Prometheus metrics.
type MetricMapping struct {
	// RedisDB is the SONiC Redis database name (e.g. "STATE_DB", "COUNTERS_DB").
	RedisDB string `json:"redis_db"`
	// KeyPattern is the Redis KEYS glob pattern (e.g. "TRANSCEIVER_INFO|*").
	KeyPattern string `json:"key_pattern"`
	// KeySeparator is the character separating the table prefix from the key suffix.
	// Defaults to "|".
	KeySeparator string `json:"key_separator,omitempty"`
	// KeyResolver is the name of a Redis hash that maps logical names to key suffixes.
	// Used for COUNTERS_DB where keys are OIDs resolved via COUNTERS_PORT_NAME_MAP.
	KeyResolver string `json:"key_resolver,omitempty"`
	// Fields defines the field-to-metric mappings for each matched key.
	Fields []FieldMapping `json:"fields"`
}

// FieldMapping defines how a Redis hash field maps to a Prometheus metric.
type FieldMapping struct {
	// Field is a specific Redis hash field name. Mutually exclusive with FieldPattern.
	Field string `json:"field,omitempty"`
	// FieldPattern iterates all fields when set to "*". Mutually exclusive with Field.
	FieldPattern string `json:"field_pattern,omitempty"`
	// Metric is the Prometheus metric name.
	Metric string `json:"metric"`
	// Type is the Prometheus metric type: "gauge" or "counter".
	Type string `json:"type"`
	// Help is the metric help string.
	Help string `json:"help,omitempty"`
	// Value is a fixed metric value (e.g. 1 for _info pattern). When set, the Redis
	// field value is ignored and this value is used instead.
	Value *float64 `json:"value,omitempty"`
	// Labels maps Prometheus label names to value templates.
	// Templates: "$key_suffix", "$port_name", "$field_name", "$<hash_field>", or literal.
	Labels map[string]string `json:"labels,omitempty"`
	// Transform defines optional value transformations.
	Transform *Transform `json:"transform,omitempty"`
}

// Transform defines value transformations for a field mapping.
type Transform struct {
	// Map converts string field values to float64 (e.g. {"up": 1, "down": 0}).
	Map map[string]float64 `json:"map,omitempty"`
	// ParseThresholdField enables special DOM threshold field name parsing,
	// which extracts sensor/level/direction labels from field names like "temphighalarm".
	ParseThresholdField bool `json:"parse_threshold_field,omitempty"`
	// RegexCapture matches field names against a regex and extracts capture groups as labels.
	RegexCapture *RegexCapture `json:"regex_capture,omitempty"`
	// DOMFlagSeverity computes a severity rollup (0=ok, 1=warning, 2=alarm) from all hash fields.
	DOMFlagSeverity bool `json:"dom_flag_severity,omitempty"`
	// Histogram maps upper bounds to Redis field names, emitting a Prometheus histogram.
	Histogram *HistogramBuckets `json:"histogram,omitempty"`
}

// RegexCapture defines a regex-based field name matching transform.
// The pattern must use Go named capture groups (?P<name>...) to define label names.
// For example, "^rx(?P<lane>\\d+)power$" matches "rx1power" and produces label lane="1".
type RegexCapture struct {
	// Pattern is a Go regex with named capture groups (e.g. "^rx(?P<lane>\\d+)power$").
	Pattern string `json:"pattern"`
}

// HistogramBuckets defines a histogram transform that maps Redis field names to
// Prometheus histogram bucket upper bounds.
type HistogramBuckets struct {
	// Buckets maps upper bounds (in bytes, seconds, etc.) to Redis hash field names.
	// Values are read, parsed as uint64, and accumulated into cumulative histogram buckets.
	Buckets map[float64]string `json:"buckets"`
}

// UnmarshalJSON implements custom JSON unmarshaling for HistogramBuckets.
// sigs.k8s.io/yaml converts YAML→JSON, so numeric YAML keys become JSON string keys.
// This method parses those string keys back to float64.
func (hb *HistogramBuckets) UnmarshalJSON(data []byte) error {
	var raw struct {
		Buckets map[string]string `json:"buckets"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	hb.Buckets = make(map[float64]string, len(raw.Buckets))
	for k, v := range raw.Buckets {
		f, err := strconv.ParseFloat(k, 64)
		if err != nil {
			return fmt.Errorf("histogram bucket key %q is not a valid number: %w", k, err)
		}
		hb.Buckets[f] = v
	}
	return nil
}

// effectiveSeparator returns the key separator, defaulting to "|".
func (m *MetricMapping) effectiveSeparator() string {
	if m.KeySeparator != "" {
		return m.KeySeparator
	}
	return "|"
}

// LoadConfig loads a MetricsConfig from a YAML file path.
func LoadConfig(path string) (*MetricsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading metrics config %s: %w", path, err)
	}
	return parseConfig(data)
}

// DefaultConfig returns the built-in default metrics configuration.
func DefaultConfig() (*MetricsConfig, error) {
	data, err := defaultConfigFS.ReadFile("default_config.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading embedded default config: %w", err)
	}
	return parseConfig(data)
}

func parseConfig(data []byte) (*MetricsConfig, error) {
	var cfg MetricsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing metrics config: %w", err)
	}
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func validateConfig(cfg *MetricsConfig) error {
	for i, m := range cfg.Metrics {
		if m.RedisDB == "" {
			return fmt.Errorf("metrics[%d]: redis_db is required", i)
		}
		if m.KeyPattern == "" {
			return fmt.Errorf("metrics[%d]: key_pattern is required", i)
		}
		for j, f := range m.Fields {
			if f.Metric == "" {
				return fmt.Errorf("metrics[%d].fields[%d]: metric is required", i, j)
			}
			if f.Type != metricTypeGauge && f.Type != metricTypeCounter && f.Type != metricTypeHistogram {
				return fmt.Errorf("metrics[%d].fields[%d]: type must be 'gauge', 'counter', or 'histogram', got %q", i, j, f.Type)
			}
			if f.Type == metricTypeHistogram {
				if f.Transform == nil || f.Transform.Histogram == nil || len(f.Transform.Histogram.Buckets) == 0 {
					return fmt.Errorf("metrics[%d].fields[%d]: histogram type requires transform.histogram.buckets", i, j)
				}
			}
			if f.Field != "" && f.FieldPattern != "" {
				return fmt.Errorf("metrics[%d].fields[%d]: field and field_pattern are mutually exclusive", i, j)
			}
			if f.Transform != nil && f.Transform.RegexCapture != nil {
				rc := f.Transform.RegexCapture
				if rc.Pattern == "" {
					return fmt.Errorf("metrics[%d].fields[%d]: regex_capture.pattern is required", i, j)
				}
				re, err := regexp.Compile(rc.Pattern)
				if err != nil {
					return fmt.Errorf("metrics[%d].fields[%d]: regex_capture.pattern is invalid: %w", i, j, err)
				}
				if re.NumSubexp() == 0 {
					return fmt.Errorf("metrics[%d].fields[%d]: regex_capture.pattern must have at least one named capture group", i, j)
				}
				for idx, name := range re.SubexpNames()[1:] {
					if name == "" {
						return fmt.Errorf("metrics[%d].fields[%d]: regex_capture.pattern group %d is unnamed; use (?P<label_name>...) syntax", i, j, idx+1)
					}
				}
			}
		}
	}
	return nil
}

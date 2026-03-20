// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// ConfigCollector is a generic Prometheus collector driven by a MetricMapping.
// It reads Redis keys matching a pattern and emits metrics based on field mappings.
type ConfigCollector struct {
	connector RedisConnector
	mapping   MetricMapping

	// descs holds one descriptor per unique metric name.
	descs map[string]*prometheus.Desc
	// compiledRegex holds pre-compiled regexes keyed by field mapping index.
	compiledRegex map[int]*regexp.Regexp
}

// NewConfigCollector creates a collector from a MetricMapping configuration entry.
func NewConfigCollector(connector RedisConnector, mapping MetricMapping) *ConfigCollector {
	descs := make(map[string]*prometheus.Desc)
	compiledRegex := make(map[int]*regexp.Regexp)

	for i, f := range mapping.Fields {
		if _, exists := descs[f.Metric]; exists {
			continue
		}
		labels := sortedLabelNames(f, mapping.KeyResolver != "")
		// If this field uses parse_threshold_field, additional labels are added dynamically
		if f.Transform != nil && f.Transform.ParseThresholdField {
			labels = appendUnique(labels, "sensor", "level", "direction")
		}
		// If this field uses regex_capture, append capture labels and pre-compile regex
		if f.Transform != nil && f.Transform.RegexCapture != nil {
			re := regexp.MustCompile(f.Transform.RegexCapture.Pattern)
			compiledRegex[i] = re
			// Extract label names from named capture groups
			for _, name := range re.SubexpNames()[1:] {
				labels = appendUnique(labels, name)
			}
		}
		descs[f.Metric] = prometheus.NewDesc(f.Metric, f.Help, labels, nil)
	}

	return &ConfigCollector{
		connector:     connector,
		mapping:       mapping,
		descs:         descs,
		compiledRegex: compiledRegex,
	}
}

func (c *ConfigCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descs {
		ch <- d
	}
}

func (c *ConfigCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
	defer cancel()

	client, err := c.connector.Connect(c.mapping.RedisDB)
	if err != nil {
		log.Printf("ConfigCollector[%s]: failed to connect to %s: %v",
			c.mapping.KeyPattern, c.mapping.RedisDB, err)
		return
	}

	sep := c.mapping.effectiveSeparator()

	// If key_resolver is set, resolve names→keys first
	var resolverMap map[string]string // keyInRedis → resolvedName
	if c.mapping.KeyResolver != "" {
		nameToOID, err := client.HGetAll(ctx, c.mapping.KeyResolver).Result()
		if err != nil {
			log.Printf("ConfigCollector[%s]: failed to read resolver %s: %v",
				c.mapping.KeyPattern, c.mapping.KeyResolver, err)
			return
		}
		// Build reverse map: "COUNTERS:oid:..." → "Ethernet0"
		resolverMap = make(map[string]string, len(nameToOID))
		prefix := strings.SplitN(c.mapping.KeyPattern, "*", 2)[0] // "COUNTERS:"
		for name, oid := range nameToOID {
			resolverMap[prefix+oid] = name
		}
	}

	// Fetch all matching keys
	keys, err := client.Keys(ctx, c.mapping.KeyPattern).Result()
	if err != nil {
		log.Printf("ConfigCollector[%s]: failed to list keys: %v",
			c.mapping.KeyPattern, err)
		return
	}

	data := batchHGetAll(ctx, client, keys)

	for _, key := range keys {
		fields, ok := data[key]
		if !ok {
			continue
		}
		keySuffix := extractKeySuffix(key, sep)

		// Resolved port name (for key_resolver)
		portName := ""
		if resolverMap != nil {
			portName, ok = resolverMap[key]
			if !ok {
				continue // skip keys not in the resolver map
			}
		}

		for fi, fm := range c.mapping.Fields {
			// dom_flag_severity operates on the whole hash, not per-field
			if fm.Transform != nil && fm.Transform.DOMFlagSeverity {
				desc := c.descs[fm.Metric]
				metricType := prometheus.GaugeValue
				if fm.Type == metricTypeCounter {
					metricType = prometheus.CounterValue
				}
				severity := domFlagSeverity(fields)
				labels := resolveLabels(fm.Labels, keySuffix, portName, "", fields)
				ch <- prometheus.MustNewConstMetric(desc, metricType, severity, labels...)
				continue
			}

			if fm.FieldPattern == "*" {
				// Iterate all fields
				c.collectAllFields(ch, fi, fm, fields, keySuffix, portName)
			} else if fm.Field != "" {
				// Specific field
				c.collectField(ch, fi, fm, fm.Field, fields, keySuffix, portName)
			} else {
				// No field specified — emit using fixed value or labels from hash fields
				c.collectField(ch, fi, fm, "", fields, keySuffix, portName)
			}
		}
	}
}

func (c *ConfigCollector) collectAllFields(
	ch chan<- prometheus.Metric,
	fieldIdx int,
	fm FieldMapping,
	hashFields map[string]string,
	keySuffix, portName string,
) {
	for fieldName, fieldVal := range hashFields {
		c.collectFieldEntry(ch, fieldIdx, fm, fieldName, fieldVal, hashFields, keySuffix, portName)
	}
}

func (c *ConfigCollector) collectField(
	ch chan<- prometheus.Metric,
	fieldIdx int,
	fm FieldMapping,
	fieldName string,
	hashFields map[string]string,
	keySuffix, portName string,
) {
	fieldVal := ""
	if fieldName != "" {
		var ok bool
		fieldVal, ok = hashFields[fieldName]
		if !ok {
			return
		}
	}
	c.collectFieldEntry(ch, fieldIdx, fm, fieldName, fieldVal, hashFields, keySuffix, portName)
}

func (c *ConfigCollector) collectFieldEntry(
	ch chan<- prometheus.Metric,
	fieldIdx int,
	fm FieldMapping,
	fieldName, fieldVal string,
	hashFields map[string]string,
	keySuffix, portName string,
) {
	desc := c.descs[fm.Metric]
	metricType := prometheus.GaugeValue
	if fm.Type == "counter" {
		metricType = prometheus.CounterValue
	}

	// Handle regex_capture transform — filter by field name and extract capture group labels.
	// Does NOT determine the value; falls through to the value resolution below.
	var captureLabels []string
	if fm.Transform != nil && fm.Transform.RegexCapture != nil {
		re := c.compiledRegex[fieldIdx]
		m := re.FindStringSubmatch(fieldName)
		if m == nil {
			return // field doesn't match, skip
		}
		captureLabels = append(captureLabels, m[1:]...)
	}

	// Handle parse_threshold_field transform
	if fm.Transform != nil && fm.Transform.ParseThresholdField {
		parsed := parseThresholdField(fieldName)
		if parsed == nil {
			return
		}
		v, err := strconv.ParseFloat(fieldVal, 64)
		if err != nil {
			return
		}
		labels := resolveLabels(fm.Labels, keySuffix, portName, fieldName, hashFields)
		labels = append(labels, parsed.Sensor, parsed.Level, parsed.Direction)
		ch <- prometheus.MustNewConstMetric(desc, metricType, v, labels...)
		return
	}

	// Determine value
	var v float64
	if fm.Value != nil {
		v = *fm.Value
	} else if fm.Transform != nil && fm.Transform.Map != nil {
		mapped, ok := fm.Transform.Map[fieldVal]
		if !ok {
			return
		}
		v = mapped
	} else {
		var err error
		v, err = strconv.ParseFloat(fieldVal, 64)
		if err != nil {
			return
		}
	}

	labels := resolveLabels(fm.Labels, keySuffix, portName, fieldName, hashFields)
	labels = append(labels, captureLabels...)
	ch <- prometheus.MustNewConstMetric(desc, metricType, v, labels...)
}

// resolveLabels resolves label value templates into concrete values, returning
// them in sorted label-name order (matching the desc's label order).
func resolveLabels(
	labelTemplates map[string]string,
	keySuffix, portName, fieldName string,
	hashFields map[string]string,
) []string {
	// Sort label names to match prometheus.Desc label order
	names := make([]string, 0, len(labelTemplates))
	for name := range labelTemplates {
		names = append(names, name)
	}
	sort.Strings(names)

	values := make([]string, 0, len(names))
	for _, name := range names {
		tmpl := labelTemplates[name]
		values = append(values, resolveTemplate(tmpl, keySuffix, portName, fieldName, hashFields))
	}
	return values
}

// resolveTemplate resolves a single label value template.
func resolveTemplate(tmpl, keySuffix, portName, fieldName string, hashFields map[string]string) string {
	if !strings.HasPrefix(tmpl, "$") {
		return tmpl // literal value
	}
	varName := tmpl[1:]
	switch varName {
	case "key_suffix":
		return keySuffix
	case "port_name":
		return portName
	case "field_name":
		return fieldName
	default:
		// Treat as a hash field reference (e.g. "$vendor_name")
		return hashFields[varName]
	}
}

// sortedLabelNames extracts label names from a FieldMapping in sorted order.
func sortedLabelNames(f FieldMapping, _ bool) []string {
	names := make([]string, 0, len(f.Labels))
	for name := range f.Labels {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// appendUnique appends items to a slice only if they're not already present.
func appendUnique(slice []string, items ...string) []string {
	set := make(map[string]bool, len(slice))
	for _, s := range slice {
		set[s] = true
	}
	for _, item := range items {
		if !set[item] {
			slice = append(slice, item)
			set[item] = true
		}
	}
	return slice
}

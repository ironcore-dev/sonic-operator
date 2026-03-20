// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import "strings"

// thresholdField represents a parsed DOM threshold field name.
type thresholdField struct {
	Sensor    string // e.g. "temperature", "voltage", "rx_power", "tx_bias", "tx_power"
	Level     string // "alarm" or "warning"
	Direction string // "high" or "low"
}

// thresholdPrefixes maps field name prefixes to sensor names and direction.
// SONiC TRANSCEIVER_DOM_THRESHOLD field names follow the pattern:
//
//	{sensor_prefix}{direction}{level}
//
// e.g. "temphighalarm", "vcclowwarning", "rxpowerhighwarning", "txbiaslowalarm"
var thresholdPrefixes = []struct {
	prefix    string
	sensor    string
	direction string
}{
	{"temphigh", "temperature", "high"},
	{"templow", "temperature", "low"},
	{"vcchigh", "voltage", "high"},
	{"vcclow", "voltage", "low"},
	{"rxpowerhigh", "rx_power", "high"},
	{"rxpowerlow", "rx_power", "low"},
	{"txbiashigh", "tx_bias", "high"},
	{"txbiaslow", "tx_bias", "low"},
	{"txpowerhigh", "tx_power", "high"},
	{"txpowerlow", "tx_power", "low"},
}

// parseThresholdField parses a SONiC DOM threshold field name into its components.
// Returns nil if the field name is not recognized.
func parseThresholdField(fieldName string) *thresholdField {
	lower := strings.ToLower(fieldName)
	for _, tp := range thresholdPrefixes {
		if !strings.HasPrefix(lower, tp.prefix) {
			continue
		}
		remainder := lower[len(tp.prefix):]
		var level string
		switch remainder {
		case "alarm":
			level = "alarm"
		case "warning":
			level = "warning"
		default:
			continue
		}
		return &thresholdField{
			Sensor:    tp.sensor,
			Level:     level,
			Direction: tp.direction,
		}
	}
	return nil
}

// domFlagSeverity computes the overall severity from DOM flag fields for a single interface.
// Flag field names follow the same pattern as thresholds: {sensor_prefix}{direction}{level}.
// Values are "true" or "false" strings from Redis.
// Returns 0 (ok), 1 (warning), or 2 (alarm).
func domFlagSeverity(fields map[string]string) float64 {
	severity := 0.0
	for fieldName, val := range fields {
		if val != "true" {
			continue
		}
		parsed := parseThresholdField(fieldName)
		if parsed == nil {
			continue
		}
		switch parsed.Level {
		case "alarm":
			return 2 // alarm is highest, return immediately
		case "warning":
			severity = 1
		}
	}
	return severity
}

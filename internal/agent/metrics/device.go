// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

// VersionInfoFunc returns static device metadata (e.g. from /etc/sonic/sonic_version.yml).
// It is called on every collect as a fallback when Redis fields are empty.
type VersionInfoFunc func() (map[string]string, error)

// DeviceCollector collects device metadata and readiness metrics from CONFIG_DB,
// with optional fallback to static version info for missing fields.
type DeviceCollector struct {
	connector   RedisConnector
	versionInfo VersionInfoFunc

	infoDesc  *prometheus.Desc
	readyDesc *prometheus.Desc
}

// NewDeviceCollector creates a collector for sonic_switch_info and sonic_switch_ready.
// versionInfo is optional — when non-nil it is called to fill in missing Redis fields.
func NewDeviceCollector(connector RedisConnector, versionInfo VersionInfoFunc) *DeviceCollector {
	return &DeviceCollector{
		connector:   connector,
		versionInfo: versionInfo,
		infoDesc: prometheus.NewDesc(
			"sonic_switch_info",
			"Device metadata as labels, always 1",
			[]string{"asic", "firmware", "hwsku", "mac", "platform"},
			nil,
		),
		readyDesc: prometheus.NewDesc(
			"sonic_switch_ready",
			"Whether the switch is ready (1) or not (0)",
			nil,
			nil,
		),
	}
}

func (c *DeviceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc
	ch <- c.readyDesc
}

func (c *DeviceCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
	defer cancel()

	configDB, err := c.connector.Connect("CONFIG_DB")
	if err != nil {
		log.Printf("DeviceCollector: failed to connect to CONFIG_DB: %v", err)
		ch <- prometheus.MustNewConstMetric(c.readyDesc, prometheus.GaugeValue, 0)
		return
	}

	fields, err := configDB.HGetAll(ctx, "DEVICE_METADATA|localhost").Result()
	if err != nil {
		log.Printf("DeviceCollector: failed to read DEVICE_METADATA: %v", err)
		ch <- prometheus.MustNewConstMetric(c.readyDesc, prometheus.GaugeValue, 0)
		return
	}

	mac := fields["mac"]
	if mac == "" {
		ch <- prometheus.MustNewConstMetric(c.readyDesc, prometheus.GaugeValue, 0)
		return
	}

	hwsku := fields["hwsku"]
	platform := fields["platform"]
	firmware := fields["sonic_os_version"]
	asic := fields["asic_type"]

	// Fall back to static version info for missing fields
	if c.versionInfo != nil && (hwsku == "" || firmware == "" || asic == "") {
		info, err := c.versionInfo()
		if err != nil {
			log.Printf("DeviceCollector: failed to read version info: %v", err)
		} else {
			if hwsku == "" {
				hwsku = info["hwsku"]
			}
			if firmware == "" {
				firmware = info["sonic_os_version"]
			}
			if asic == "" {
				asic = info["asic_type"]
			}
		}
	}

	ch <- prometheus.MustNewConstMetric(
		c.infoDesc,
		prometheus.GaugeValue,
		1,
		asic,
		firmware,
		hwsku,
		mac,
		platform,
	)
	ch <- prometheus.MustNewConstMetric(c.readyDesc, prometheus.GaugeValue, 1)
}

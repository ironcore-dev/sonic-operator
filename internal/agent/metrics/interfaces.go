// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

// InterfaceCollector collects interface state and count metrics.
type InterfaceCollector struct {
	connector RedisConnector

	operStateDesc      *prometheus.Desc
	adminStateDesc     *prometheus.Desc
	interfaceTotalDesc *prometheus.Desc
	portsTotalDesc     *prometheus.Desc
}

// NewInterfaceCollector creates a collector for interface state and count metrics.
func NewInterfaceCollector(connector RedisConnector) *InterfaceCollector {
	return &InterfaceCollector{
		connector: connector,
		operStateDesc: prometheus.NewDesc(
			"sonic_switch_interface_oper_state",
			"Operational state of the interface (1=up, 0=down)",
			[]string{"interface"},
			nil,
		),
		adminStateDesc: prometheus.NewDesc(
			"sonic_switch_interface_admin_state",
			"Admin state of the interface (1=up, 0=down)",
			[]string{"interface"},
			nil,
		),
		interfaceTotalDesc: prometheus.NewDesc(
			"sonic_switch_interfaces_total",
			"Number of interfaces by operational status",
			[]string{"operational_status"},
			nil,
		),
		portsTotalDesc: prometheus.NewDesc(
			"sonic_switch_ports_total",
			"Total number of physical ports",
			nil,
			nil,
		),
	}
}

func (c *InterfaceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.operStateDesc
	ch <- c.adminStateDesc
	ch <- c.interfaceTotalDesc
	ch <- c.portsTotalDesc
}

func (c *InterfaceCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
	defer cancel()

	configDB, err := c.connector.Connect("CONFIG_DB")
	if err != nil {
		log.Printf("InterfaceCollector: failed to connect to CONFIG_DB: %v", err)
		return
	}

	applDB, err := c.connector.Connect("APPL_DB")
	if err != nil {
		log.Printf("InterfaceCollector: failed to connect to APPL_DB: %v", err)
		return
	}

	// Get all configured port keys from CONFIG_DB.
	// NOTE: Redis KEYS is O(N) but acceptable here — SONiC switches have a bounded
	// number of physical ports (typically <256).
	portKeys, err := configDB.Keys(ctx, "PORT|*").Result()
	if err != nil {
		log.Printf("InterfaceCollector: failed to list PORT keys: %v", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.portsTotalDesc, prometheus.GaugeValue, float64(len(portKeys)))

	// Build state key list and fetch all state hashes in one pipeline
	ifaceNames := make([]string, 0, len(portKeys))
	stateKeys := make([]string, 0, len(portKeys))
	for _, key := range portKeys {
		name := extractKeySuffix(key, "|")
		ifaceNames = append(ifaceNames, name)
		stateKeys = append(stateKeys, "PORT_TABLE|"+name)
	}

	stateData := batchHGetAll(ctx, applDB, stateKeys)

	upCount := 0
	downCount := 0

	for i, name := range ifaceNames {
		stateKey := stateKeys[i]
		fields := stateData[stateKey]

		operUp := fields["oper_status"] == "up"
		adminUp := fields["admin_status"] == "up"

		operVal := 0.0
		if operUp {
			operVal = 1.0
			upCount++
		} else {
			downCount++
		}

		adminVal := 0.0
		if adminUp {
			adminVal = 1.0
		}

		ch <- prometheus.MustNewConstMetric(c.operStateDesc, prometheus.GaugeValue, operVal, name)
		ch <- prometheus.MustNewConstMetric(c.adminStateDesc, prometheus.GaugeValue, adminVal, name)
	}

	ch <- prometheus.MustNewConstMetric(c.interfaceTotalDesc, prometheus.GaugeValue, float64(upCount), "up")
	ch <- prometheus.MustNewConstMetric(c.interfaceTotalDesc, prometheus.GaugeValue, float64(downCount), "down")
}

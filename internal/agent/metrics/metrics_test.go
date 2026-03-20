// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
)

// mockConnector implements RedisConnector for testing using redismock.
type mockConnector struct {
	clients map[string]*redis.Client
	mocks   map[string]redismock.ClientMock
}

func newMockConnector(dbNames ...string) *mockConnector {
	mc := &mockConnector{
		clients: make(map[string]*redis.Client),
		mocks:   make(map[string]redismock.ClientMock),
	}
	for _, name := range dbNames {
		client, mock := redismock.NewClientMock()
		mc.clients[name] = client
		mc.mocks[name] = mock
	}
	return mc
}

func (mc *mockConnector) Connect(dbName string) (*redis.Client, error) {
	c, ok := mc.clients[dbName]
	if !ok {
		return nil, fmt.Errorf("no mock for database %s", dbName)
	}
	return c, nil
}

func (mc *mockConnector) expectationsMet(t *testing.T) {
	t.Helper()
	for name, mock := range mc.mocks {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations for %s: %v", name, err)
		}
	}
}

// --- Device Collector Tests ---

func TestDeviceCollector(t *testing.T) {
	mc := newMockConnector("CONFIG_DB")
	mc.mocks["CONFIG_DB"].ExpectHGetAll("DEVICE_METADATA|localhost").SetVal(map[string]string{
		"mac":              "aa:bb:cc:dd:ee:ff",
		"hwsku":            "Accton-AS7726-32X",
		"platform":         "x86_64-accton_as7726_32x-r0",
		"sonic_os_version": "4.2.0",
		"asic_type":        "broadcom",
	})

	collector := NewDeviceCollector(mc, nil)
	expected := `
		# HELP sonic_switch_info Device metadata as labels, always 1
		# TYPE sonic_switch_info gauge
		sonic_switch_info{asic="broadcom",firmware="4.2.0",hwsku="Accton-AS7726-32X",mac="aa:bb:cc:dd:ee:ff",platform="x86_64-accton_as7726_32x-r0"} 1
		# HELP sonic_switch_ready Whether the switch is ready (1) or not (0)
		# TYPE sonic_switch_ready gauge
		sonic_switch_ready 1
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("DeviceCollector mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

func TestDeviceCollectorFallback(t *testing.T) {
	mc := newMockConnector("CONFIG_DB")
	mc.mocks["CONFIG_DB"].ExpectHGetAll("DEVICE_METADATA|localhost").SetVal(map[string]string{
		"mac":      "aa:bb:cc:dd:ee:ff",
		"hwsku":    "Accton-AS7726-32X",
		"platform": "x86_64-accton_as7726_32x-r0",
		// No sonic_os_version or asic_type in Redis
	})

	versionInfo := func() (map[string]string, error) {
		return map[string]string{
			"sonic_os_version": "4.4.0",
			"asic_type":        "broadcom",
		}, nil
	}

	collector := NewDeviceCollector(mc, versionInfo)
	expected := `
		# HELP sonic_switch_info Device metadata as labels, always 1
		# TYPE sonic_switch_info gauge
		sonic_switch_info{asic="broadcom",firmware="4.4.0",hwsku="Accton-AS7726-32X",mac="aa:bb:cc:dd:ee:ff",platform="x86_64-accton_as7726_32x-r0"} 1
		# HELP sonic_switch_ready Whether the switch is ready (1) or not (0)
		# TYPE sonic_switch_ready gauge
		sonic_switch_ready 1
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("DeviceCollector fallback mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

func TestDeviceCollectorNotReady(t *testing.T) {
	mc := newMockConnector("CONFIG_DB")
	mc.mocks["CONFIG_DB"].ExpectHGetAll("DEVICE_METADATA|localhost").SetVal(map[string]string{
		// No "mac" field → not ready
		"hwsku": "Accton-AS7726-32X",
	})

	collector := NewDeviceCollector(mc, nil)
	expected := `
		# HELP sonic_switch_ready Whether the switch is ready (1) or not (0)
		# TYPE sonic_switch_ready gauge
		sonic_switch_ready 0
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "sonic_switch_ready"); err != nil {
		t.Errorf("DeviceCollector not-ready mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

// --- Interface Collector Tests ---

func TestInterfaceCollector(t *testing.T) {
	mc := newMockConnector("CONFIG_DB", "STATE_DB")

	mc.mocks["CONFIG_DB"].ExpectKeys("PORT|*").SetVal([]string{
		"PORT|Ethernet0", "PORT|Ethernet4",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("PORT_TABLE|Ethernet0").SetVal(map[string]string{
		"netdev_oper_status": "up",
		"admin_status":       "up",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("PORT_TABLE|Ethernet4").SetVal(map[string]string{
		"netdev_oper_status": "down",
		"admin_status":       "up",
	})

	collector := NewInterfaceCollector(mc)

	expected := `
		# HELP sonic_switch_ports_total Total number of physical ports
		# TYPE sonic_switch_ports_total gauge
		sonic_switch_ports_total 2
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "sonic_switch_ports_total"); err != nil {
		t.Errorf("InterfaceCollector ports_total mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

func TestInterfaceCollectorInterfaceTotals(t *testing.T) {
	mc := newMockConnector("CONFIG_DB", "STATE_DB")

	mc.mocks["CONFIG_DB"].ExpectKeys("PORT|*").SetVal([]string{
		"PORT|Ethernet0", "PORT|Ethernet4", "PORT|Ethernet8",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("PORT_TABLE|Ethernet0").SetVal(map[string]string{
		"netdev_oper_status": "up",
		"admin_status":       "up",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("PORT_TABLE|Ethernet4").SetVal(map[string]string{
		"netdev_oper_status": "up",
		"admin_status":       "up",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("PORT_TABLE|Ethernet8").SetVal(map[string]string{
		"netdev_oper_status": "down",
		"admin_status":       "down",
	})

	collector := NewInterfaceCollector(mc)
	expected := `
		# HELP sonic_switch_interfaces_total Number of interfaces by operational status
		# TYPE sonic_switch_interfaces_total gauge
		sonic_switch_interfaces_total{operational_status="up"} 2
		sonic_switch_interfaces_total{operational_status="down"} 1
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "sonic_switch_interfaces_total"); err != nil {
		t.Errorf("InterfaceCollector interfaces_total mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

// --- DOM Sensor Collector Tests (config-driven) ---

func TestConfigCollectorDOMSensors(t *testing.T) {
	mc := newMockConnector("STATE_DB")

	mc.mocks["STATE_DB"].ExpectKeys("TRANSCEIVER_DOM_SENSOR|*").SetVal([]string{
		"TRANSCEIVER_DOM_SENSOR|Ethernet0",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("TRANSCEIVER_DOM_SENSOR|Ethernet0").SetVal(map[string]string{
		"temperature": "32.5",
		"voltage":     "3.31",
		"rx1power":    "-8.42",
		"rx2power":    "-7.50",
		"tx1bias":     "6.75",
		"tx2bias":     "6.80",
	})

	// Temperature — simple field
	tempMapping := MetricMapping{
		RedisDB:      "STATE_DB",
		KeyPattern:   "TRANSCEIVER_DOM_SENSOR|*",
		KeySeparator: "|",
		Fields: []FieldMapping{
			{
				Field:  "temperature",
				Metric: "sonic_switch_transceiver_dom_temperature_celsius",
				Type:   "gauge",
				Help:   "Transceiver temperature in Celsius",
				Labels: map[string]string{"interface": "$key_suffix"},
			},
			{
				Field:  "voltage",
				Metric: "sonic_switch_transceiver_dom_voltage_volts",
				Type:   "gauge",
				Help:   "Transceiver supply voltage in Volts",
				Labels: map[string]string{"interface": "$key_suffix"},
			},
			{
				FieldPattern: "*",
				Metric:       "sonic_switch_transceiver_dom_rx_power_dbm",
				Type:         "gauge",
				Help:         "Transceiver RX power in dBm",
				Labels:       map[string]string{"interface": "$key_suffix"},
				Transform: &Transform{
					RegexCapture: &RegexCapture{
						Pattern: `^rx(?P<lane>\d+)power$`,
					},
				},
			},
			{
				FieldPattern: "*",
				Metric:       "sonic_switch_transceiver_dom_tx_bias_milliamps",
				Type:         "gauge",
				Help:         "Transceiver TX bias current in milliamps",
				Labels:       map[string]string{"interface": "$key_suffix"},
				Transform: &Transform{
					RegexCapture: &RegexCapture{
						Pattern: `^tx(?P<lane>\d+)bias$`,
					},
				},
			},
		},
	}

	collector := NewConfigCollector(mc, tempMapping)
	expected := `
		# HELP sonic_switch_transceiver_dom_temperature_celsius Transceiver temperature in Celsius
		# TYPE sonic_switch_transceiver_dom_temperature_celsius gauge
		sonic_switch_transceiver_dom_temperature_celsius{interface="Ethernet0"} 32.5
		# HELP sonic_switch_transceiver_dom_voltage_volts Transceiver supply voltage in Volts
		# TYPE sonic_switch_transceiver_dom_voltage_volts gauge
		sonic_switch_transceiver_dom_voltage_volts{interface="Ethernet0"} 3.31
		# HELP sonic_switch_transceiver_dom_rx_power_dbm Transceiver RX power in dBm
		# TYPE sonic_switch_transceiver_dom_rx_power_dbm gauge
		sonic_switch_transceiver_dom_rx_power_dbm{interface="Ethernet0",lane="1"} -8.42
		sonic_switch_transceiver_dom_rx_power_dbm{interface="Ethernet0",lane="2"} -7.5
		# HELP sonic_switch_transceiver_dom_tx_bias_milliamps Transceiver TX bias current in milliamps
		# TYPE sonic_switch_transceiver_dom_tx_bias_milliamps gauge
		sonic_switch_transceiver_dom_tx_bias_milliamps{interface="Ethernet0",lane="1"} 6.75
		sonic_switch_transceiver_dom_tx_bias_milliamps{interface="Ethernet0",lane="2"} 6.8
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("ConfigCollector DOM sensors mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

// --- Threshold Field Parsing Tests ---

func TestParseThresholdField(t *testing.T) {
	tests := []struct {
		input    string
		expected *thresholdField
	}{
		{"temphighalarm", &thresholdField{"temperature", "alarm", "high"}},
		{"temphighwarning", &thresholdField{"temperature", "warning", "high"}},
		{"templowalarm", &thresholdField{"temperature", "alarm", "low"}},
		{"templowwarning", &thresholdField{"temperature", "warning", "low"}},
		{"vcchighalarm", &thresholdField{"voltage", "alarm", "high"}},
		{"vcclowwarning", &thresholdField{"voltage", "warning", "low"}},
		{"rxpowerhighalarm", &thresholdField{"rx_power", "alarm", "high"}},
		{"rxpowerlowwarning", &thresholdField{"rx_power", "warning", "low"}},
		{"txbiashighalarm", &thresholdField{"tx_bias", "alarm", "high"}},
		{"txbiaslowalarm", &thresholdField{"tx_bias", "alarm", "low"}},
		{"txpowerhighalarm", &thresholdField{"tx_power", "alarm", "high"}},
		{"txpowerlowwarning", &thresholdField{"tx_power", "warning", "low"}},
		{"unknownfield", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseThresholdField(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected %+v, got nil", tt.expected)
			}
			if *result != *tt.expected {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

// --- DOM Flag Severity Tests ---

func TestDomFlagSeverity(t *testing.T) {
	tests := []struct {
		name     string
		fields   map[string]string
		expected float64
	}{
		{"all ok", map[string]string{
			"temphighalarm": "false", "temphighwarning": "false",
		}, 0},
		{"warning", map[string]string{
			"temphighalarm": "false", "temphighwarning": "true",
		}, 1},
		{"alarm", map[string]string{
			"temphighalarm": "true", "temphighwarning": "false",
		}, 2},
		{"alarm overrides warning", map[string]string{
			"temphighalarm": "true", "vcchighwarning": "true",
		}, 2},
		{"empty fields", map[string]string{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domFlagSeverity(tt.fields)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// --- DOM Flag Collector Tests (config-driven) ---

func TestConfigCollectorDOMFlagSeverity(t *testing.T) {
	mc := newMockConnector("STATE_DB")

	mc.mocks["STATE_DB"].ExpectKeys("TRANSCEIVER_DOM_FLAG|*").SetVal([]string{
		"TRANSCEIVER_DOM_FLAG|Ethernet0",
		"TRANSCEIVER_DOM_FLAG|Ethernet4",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("TRANSCEIVER_DOM_FLAG|Ethernet0").SetVal(map[string]string{
		"temphighalarm":   "false",
		"temphighwarning": "false",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("TRANSCEIVER_DOM_FLAG|Ethernet4").SetVal(map[string]string{
		"temphighalarm":   "false",
		"temphighwarning": "true",
	})

	mapping := MetricMapping{
		RedisDB:      "STATE_DB",
		KeyPattern:   "TRANSCEIVER_DOM_FLAG|*",
		KeySeparator: "|",
		Fields: []FieldMapping{
			{
				Metric: "sonic_switch_transceiver_dom_status",
				Type:   "gauge",
				Help:   "Transceiver DOM status severity (0=ok, 1=warning, 2=alarm)",
				Labels: map[string]string{"interface": "$key_suffix"},
				Transform: &Transform{
					DOMFlagSeverity: true,
				},
			},
		},
	}

	collector := NewConfigCollector(mc, mapping)
	expected := `
		# HELP sonic_switch_transceiver_dom_status Transceiver DOM status severity (0=ok, 1=warning, 2=alarm)
		# TYPE sonic_switch_transceiver_dom_status gauge
		sonic_switch_transceiver_dom_status{interface="Ethernet0"} 0
		sonic_switch_transceiver_dom_status{interface="Ethernet4"} 1
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("ConfigCollector DOM flag severity mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

// --- Config-driven Collector Tests ---

func TestConfigCollectorTransceiverInfo(t *testing.T) {
	mc := newMockConnector("STATE_DB")

	mc.mocks["STATE_DB"].ExpectKeys("TRANSCEIVER_INFO|*").SetVal([]string{
		"TRANSCEIVER_INFO|Ethernet0",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("TRANSCEIVER_INFO|Ethernet0").SetVal(map[string]string{
		"type":         "QSFP28",
		"manufacturer": "Finisar",
		"model":        "FTLX8574D3BCL",
		"serial":       "ABC1234",
	})

	one := 1.0
	mapping := MetricMapping{
		RedisDB:      "STATE_DB",
		KeyPattern:   "TRANSCEIVER_INFO|*",
		KeySeparator: "|",
		Fields: []FieldMapping{
			{
				Metric: "sonic_switch_transceiver_info",
				Type:   "gauge",
				Help:   "Transceiver static metadata as labels, always 1",
				Value:  &one,
				Labels: map[string]string{
					"interface": "$key_suffix",
					"type":      "$type",
					"vendor":    "$manufacturer",
					"model":     "$model",
					"serial":    "$serial",
				},
			},
		},
	}

	collector := NewConfigCollector(mc, mapping)
	expected := `
		# HELP sonic_switch_transceiver_info Transceiver static metadata as labels, always 1
		# TYPE sonic_switch_transceiver_info gauge
		sonic_switch_transceiver_info{interface="Ethernet0",model="FTLX8574D3BCL",serial="ABC1234",type="QSFP28",vendor="Finisar"} 1
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("ConfigCollector transceiver info mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

func TestConfigCollectorTransceiverStatus(t *testing.T) {
	mc := newMockConnector("STATE_DB")

	mc.mocks["STATE_DB"].ExpectKeys("TRANSCEIVER_STATUS|*").SetVal([]string{
		"TRANSCEIVER_STATUS|Ethernet0",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("TRANSCEIVER_STATUS|Ethernet0").SetVal(map[string]string{
		"status":              "1",
		"error":               "N/A",
		"rxlos1":              "False",
		"rxlos2":              "True",
		"txfault1":            "False",
		"txfault2":            "False",
		"tx1disable":          "False",
		"tx2disable":          "False",
		"tx_disabled_channel": "0",
	})

	mapping := MetricMapping{
		RedisDB:      "STATE_DB",
		KeyPattern:   "TRANSCEIVER_STATUS|*",
		KeySeparator: "|",
		Fields: []FieldMapping{
			{
				FieldPattern: "*",
				Metric:       "sonic_switch_transceiver_rxlos",
				Type:         "gauge",
				Help:         "Transceiver RX loss of signal (1=loss, 0=ok)",
				Labels:       map[string]string{"interface": "$key_suffix"},
				Transform: &Transform{
					RegexCapture: &RegexCapture{
						Pattern: `^rxlos(?P<lane>\d+)$`,
					},
					Map: map[string]float64{"True": 1, "False": 0},
				},
			},
			{
				FieldPattern: "*",
				Metric:       "sonic_switch_transceiver_txfault",
				Type:         "gauge",
				Help:         "Transceiver TX fault (1=fault, 0=ok)",
				Labels:       map[string]string{"interface": "$key_suffix"},
				Transform: &Transform{
					RegexCapture: &RegexCapture{
						Pattern: `^txfault(?P<lane>\d+)$`,
					},
					Map: map[string]float64{"True": 1, "False": 0},
				},
			},
		},
	}

	collector := NewConfigCollector(mc, mapping)
	expected := `
		# HELP sonic_switch_transceiver_rxlos Transceiver RX loss of signal (1=loss, 0=ok)
		# TYPE sonic_switch_transceiver_rxlos gauge
		sonic_switch_transceiver_rxlos{interface="Ethernet0",lane="1"} 0
		sonic_switch_transceiver_rxlos{interface="Ethernet0",lane="2"} 1
		# HELP sonic_switch_transceiver_txfault Transceiver TX fault (1=fault, 0=ok)
		# TYPE sonic_switch_transceiver_txfault gauge
		sonic_switch_transceiver_txfault{interface="Ethernet0",lane="1"} 0
		sonic_switch_transceiver_txfault{interface="Ethernet0",lane="2"} 0
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("ConfigCollector transceiver status mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

func TestConfigCollectorTemperature(t *testing.T) {
	mc := newMockConnector("STATE_DB")

	mc.mocks["STATE_DB"].ExpectKeys("TEMPERATURE_INFO|*").SetVal([]string{
		"TEMPERATURE_INFO|MB_RearMAC_temp(0x48)",
		"TEMPERATURE_INFO|CPU_temp(0x4b)",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("TEMPERATURE_INFO|MB_RearMAC_temp(0x48)").SetVal(map[string]string{
		"temperature":    "34.5",
		"high_threshold": "80.0",
		"low_threshold":  "N/A",
		"warning_status": "False",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("TEMPERATURE_INFO|CPU_temp(0x4b)").SetVal(map[string]string{
		"temperature":    "42.0",
		"high_threshold": "95.0",
		"low_threshold":  "N/A",
		"warning_status": "False",
	})

	mapping := MetricMapping{
		RedisDB:      "STATE_DB",
		KeyPattern:   "TEMPERATURE_INFO|*",
		KeySeparator: "|",
		Fields: []FieldMapping{
			{
				Field:  "temperature",
				Metric: "sonic_switch_temperature_celsius",
				Type:   "gauge",
				Help:   "Chassis temperature sensor reading in Celsius",
				Labels: map[string]string{"sensor": "$key_suffix"},
			},
			{
				Field:  "high_threshold",
				Metric: "sonic_switch_temperature_high_threshold_celsius",
				Type:   "gauge",
				Help:   "Chassis temperature sensor high threshold in Celsius",
				Labels: map[string]string{"sensor": "$key_suffix"},
			},
			{
				Field:  "warning_status",
				Metric: "sonic_switch_temperature_warning",
				Type:   "gauge",
				Help:   "Chassis temperature sensor warning status (1=warning, 0=ok)",
				Labels: map[string]string{"sensor": "$key_suffix"},
				Transform: &Transform{
					Map: map[string]float64{"True": 1, "False": 0},
				},
			},
		},
	}

	collector := NewConfigCollector(mc, mapping)
	expected := `
		# HELP sonic_switch_temperature_celsius Chassis temperature sensor reading in Celsius
		# TYPE sonic_switch_temperature_celsius gauge
		sonic_switch_temperature_celsius{sensor="CPU_temp(0x4b)"} 42
		sonic_switch_temperature_celsius{sensor="MB_RearMAC_temp(0x48)"} 34.5
		# HELP sonic_switch_temperature_high_threshold_celsius Chassis temperature sensor high threshold in Celsius
		# TYPE sonic_switch_temperature_high_threshold_celsius gauge
		sonic_switch_temperature_high_threshold_celsius{sensor="CPU_temp(0x4b)"} 95
		sonic_switch_temperature_high_threshold_celsius{sensor="MB_RearMAC_temp(0x48)"} 80
		# HELP sonic_switch_temperature_warning Chassis temperature sensor warning status (1=warning, 0=ok)
		# TYPE sonic_switch_temperature_warning gauge
		sonic_switch_temperature_warning{sensor="CPU_temp(0x4b)"} 0
		sonic_switch_temperature_warning{sensor="MB_RearMAC_temp(0x48)"} 0
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("ConfigCollector temperature mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

func TestConfigCollectorLLDPNeighborInfo(t *testing.T) {
	mc := newMockConnector("APPL_DB")

	mc.mocks["APPL_DB"].ExpectKeys("LLDP_ENTRY_TABLE:*").SetVal([]string{
		"LLDP_ENTRY_TABLE:Ethernet120",
	})
	mc.mocks["APPL_DB"].ExpectHGetAll("LLDP_ENTRY_TABLE:Ethernet120").SetVal(map[string]string{
		"lldp_rem_chassis_id":         "94:ef:97:94:65:42",
		"lldp_rem_sys_name":           "swi2-wdf4g-1",
		"lldp_rem_port_desc":          "Ethernet8",
		"lldp_rem_port_id":            "Eth3(Port3)",
		"lldp_rem_man_addr":           "240.127.1.1",
		"lldp_rem_chassis_id_subtype": "4",
		"lldp_rem_port_id_subtype":    "7",
		"lldp_rem_index":              "1",
		"lldp_rem_time_mark":          "44873750",
		"lldp_rem_sys_desc":           "SONiC Software Version: SONiC.Edgecore",
		"lldp_rem_sys_cap_supported":  "28 00",
		"lldp_rem_sys_cap_enabled":    "28 00",
	})

	one := 1.0
	mapping := MetricMapping{
		RedisDB:      "APPL_DB",
		KeyPattern:   "LLDP_ENTRY_TABLE:*",
		KeySeparator: ":",
		Fields: []FieldMapping{
			{
				Metric: "sonic_switch_interface_neighbor_info",
				Type:   "gauge",
				Help:   "LLDP neighbor metadata as labels, always 1",
				Value:  &one,
				Labels: map[string]string{
					"interface":     "$key_suffix",
					"neighbor_mac":  "$lldp_rem_chassis_id",
					"neighbor_name": "$lldp_rem_sys_name",
					"neighbor_port": "$lldp_rem_port_desc",
				},
			},
		},
	}

	collector := NewConfigCollector(mc, mapping)
	expected := `
		# HELP sonic_switch_interface_neighbor_info LLDP neighbor metadata as labels, always 1
		# TYPE sonic_switch_interface_neighbor_info gauge
		sonic_switch_interface_neighbor_info{interface="Ethernet120",neighbor_mac="94:ef:97:94:65:42",neighbor_name="swi2-wdf4g-1",neighbor_port="Ethernet8"} 1
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("ConfigCollector LLDP neighbor info mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

func TestConfigCollectorDOMThresholds(t *testing.T) {
	mc := newMockConnector("STATE_DB")

	mc.mocks["STATE_DB"].ExpectKeys("TRANSCEIVER_DOM_THRESHOLD|*").SetVal([]string{
		"TRANSCEIVER_DOM_THRESHOLD|Ethernet0",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("TRANSCEIVER_DOM_THRESHOLD|Ethernet0").SetVal(map[string]string{
		"temphighalarm":     "70.0",
		"templowalarm":      "-5.0",
		"rxpowerlowwarning": "-14.0",
	})

	mapping := MetricMapping{
		RedisDB:      "STATE_DB",
		KeyPattern:   "TRANSCEIVER_DOM_THRESHOLD|*",
		KeySeparator: "|",
		Fields: []FieldMapping{
			{
				FieldPattern: "*",
				Metric:       "sonic_switch_transceiver_dom_threshold",
				Type:         "gauge",
				Help:         "Transceiver DOM threshold value",
				Labels: map[string]string{
					"interface": "$key_suffix",
				},
				Transform: &Transform{
					ParseThresholdField: true,
				},
			},
		},
	}

	collector := NewConfigCollector(mc, mapping)
	expected := `
		# HELP sonic_switch_transceiver_dom_threshold Transceiver DOM threshold value
		# TYPE sonic_switch_transceiver_dom_threshold gauge
		sonic_switch_transceiver_dom_threshold{direction="high",interface="Ethernet0",level="alarm",sensor="temperature"} 70
		sonic_switch_transceiver_dom_threshold{direction="low",interface="Ethernet0",level="alarm",sensor="temperature"} -5
		sonic_switch_transceiver_dom_threshold{direction="low",interface="Ethernet0",level="warning",sensor="rx_power"} -14
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("ConfigCollector DOM thresholds mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

func TestConfigCollectorCounters(t *testing.T) {
	mc := newMockConnector("COUNTERS_DB")

	mc.mocks["COUNTERS_DB"].ExpectHGetAll("COUNTERS_PORT_NAME_MAP").SetVal(map[string]string{
		"Ethernet0": "oid:0x100000000003",
	})
	mc.mocks["COUNTERS_DB"].ExpectKeys("COUNTERS:*").SetVal([]string{
		"COUNTERS:oid:0x100000000003",
	})
	mc.mocks["COUNTERS_DB"].ExpectHGetAll("COUNTERS:oid:0x100000000003").SetVal(map[string]string{
		"SAI_PORT_STAT_IF_IN_OCTETS":                     "123456789",
		"SAI_PORT_STAT_IF_OUT_OCTETS":                    "987654321",
		"SAI_PORT_STAT_IF_IN_UCAST_PKTS":                 "100000",
		"SAI_PORT_STAT_IF_OUT_UCAST_PKTS":                "200000",
		"SAI_PORT_STAT_IF_IN_MULTICAST_PKTS":             "500",
		"SAI_PORT_STAT_IF_OUT_MULTICAST_PKTS":            "300",
		"SAI_PORT_STAT_IF_IN_BROADCAST_PKTS":             "50",
		"SAI_PORT_STAT_IF_OUT_BROADCAST_PKTS":            "20",
		"SAI_PORT_STAT_IF_IN_NON_UCAST_PKTS":             "550",
		"SAI_PORT_STAT_IF_OUT_NON_UCAST_PKTS":            "320",
		"SAI_PORT_STAT_IF_IN_ERRORS":                      "42",
		"SAI_PORT_STAT_IF_OUT_ERRORS":                     "0",
		"SAI_PORT_STAT_IF_IN_DISCARDS":                    "7",
		"SAI_PORT_STAT_IF_OUT_DISCARDS":                   "3",
		"SAI_PORT_STAT_IN_DROPPED_PKTS":                   "2",
		"SAI_PORT_STAT_OUT_DROPPED_PKTS":                  "1",
		"SAI_PORT_STAT_IF_IN_FEC_CORRECTABLE_FRAMES":      "1580",
		"SAI_PORT_STAT_IF_IN_FEC_NOT_CORRECTABLE_FRAMES":  "0",
		"SAI_PORT_STAT_IF_IN_FEC_SYMBOL_ERRORS":           "23",
		"SAI_PORT_STAT_IF_OUT_QLEN":                       "5",
		"SAI_PORT_STAT_PFC_0_RX_PKTS":                     "100",
		"SAI_PORT_STAT_PFC_0_TX_PKTS":                     "50",
		"SAI_PORT_STAT_ETHER_IN_PKTS_64_OCTETS":           "10000",
		"SAI_PORT_STAT_ETHER_OUT_PKTS_64_OCTETS":          "8000",
		"SAI_PORT_STAT_ETHER_STATS_UNDERSIZE_PKTS":        "0",
		"SAI_PORT_STAT_ETHER_STATS_FRAGMENTS":             "0",
		"SAI_PORT_STAT_ETHER_STATS_JABBERS":               "0",
		"SAI_PORT_STAT_IF_IN_UNKNOWN_PROTOS":              "0",
		"SAI_PORT_STAT_ETHER_RX_OVERSIZE_PKTS":            "0",
		"SAI_PORT_STAT_ETHER_TX_OVERSIZE_PKTS":            "0",
	})

	mapping := MetricMapping{
		RedisDB:      "COUNTERS_DB",
		KeyPattern:   "COUNTERS:*",
		KeySeparator: ":",
		KeyResolver:  "COUNTERS_PORT_NAME_MAP",
		Fields: []FieldMapping{
			{Field: "SAI_PORT_STAT_IF_IN_OCTETS", Metric: "sonic_switch_interface_bytes_total", Type: "counter", Help: "Total bytes transferred", Labels: map[string]string{"interface": "$port_name", "direction": "rx"}},
			{Field: "SAI_PORT_STAT_IF_OUT_OCTETS", Metric: "sonic_switch_interface_bytes_total", Type: "counter", Help: "Total bytes transferred", Labels: map[string]string{"interface": "$port_name", "direction": "tx"}},
			{Field: "SAI_PORT_STAT_IF_IN_UCAST_PKTS", Metric: "sonic_switch_interface_packets_total", Type: "counter", Help: "Total packets transferred", Labels: map[string]string{"interface": "$port_name", "direction": "rx", "type": "unicast"}},
			{Field: "SAI_PORT_STAT_IF_OUT_UCAST_PKTS", Metric: "sonic_switch_interface_packets_total", Type: "counter", Help: "Total packets transferred", Labels: map[string]string{"interface": "$port_name", "direction": "tx", "type": "unicast"}},
			{Field: "SAI_PORT_STAT_IF_IN_MULTICAST_PKTS", Metric: "sonic_switch_interface_packets_total", Type: "counter", Help: "Total packets transferred", Labels: map[string]string{"interface": "$port_name", "direction": "rx", "type": "multicast"}},
			{Field: "SAI_PORT_STAT_IF_OUT_MULTICAST_PKTS", Metric: "sonic_switch_interface_packets_total", Type: "counter", Help: "Total packets transferred", Labels: map[string]string{"interface": "$port_name", "direction": "tx", "type": "multicast"}},
			{Field: "SAI_PORT_STAT_IF_IN_BROADCAST_PKTS", Metric: "sonic_switch_interface_packets_total", Type: "counter", Help: "Total packets transferred", Labels: map[string]string{"interface": "$port_name", "direction": "rx", "type": "broadcast"}},
			{Field: "SAI_PORT_STAT_IF_OUT_BROADCAST_PKTS", Metric: "sonic_switch_interface_packets_total", Type: "counter", Help: "Total packets transferred", Labels: map[string]string{"interface": "$port_name", "direction": "tx", "type": "broadcast"}},
			{Field: "SAI_PORT_STAT_IF_IN_NON_UCAST_PKTS", Metric: "sonic_switch_interface_packets_total", Type: "counter", Help: "Total packets transferred", Labels: map[string]string{"interface": "$port_name", "direction": "rx", "type": "non_unicast"}},
			{Field: "SAI_PORT_STAT_IF_OUT_NON_UCAST_PKTS", Metric: "sonic_switch_interface_packets_total", Type: "counter", Help: "Total packets transferred", Labels: map[string]string{"interface": "$port_name", "direction": "tx", "type": "non_unicast"}},
			{Field: "SAI_PORT_STAT_IF_IN_ERRORS", Metric: "sonic_switch_interface_errors_total", Type: "counter", Help: "Total interface errors", Labels: map[string]string{"interface": "$port_name", "direction": "rx"}},
			{Field: "SAI_PORT_STAT_IF_OUT_ERRORS", Metric: "sonic_switch_interface_errors_total", Type: "counter", Help: "Total interface errors", Labels: map[string]string{"interface": "$port_name", "direction": "tx"}},
			{Field: "SAI_PORT_STAT_IF_IN_DISCARDS", Metric: "sonic_switch_interface_discards_total", Type: "counter", Help: "Total interface discards", Labels: map[string]string{"interface": "$port_name", "direction": "rx"}},
			{Field: "SAI_PORT_STAT_IF_OUT_DISCARDS", Metric: "sonic_switch_interface_discards_total", Type: "counter", Help: "Total interface discards", Labels: map[string]string{"interface": "$port_name", "direction": "tx"}},
			{Field: "SAI_PORT_STAT_IN_DROPPED_PKTS", Metric: "sonic_switch_interface_dropped_packets_total", Type: "counter", Help: "Total SAI-level dropped packets", Labels: map[string]string{"interface": "$port_name", "direction": "rx"}},
			{Field: "SAI_PORT_STAT_OUT_DROPPED_PKTS", Metric: "sonic_switch_interface_dropped_packets_total", Type: "counter", Help: "Total SAI-level dropped packets", Labels: map[string]string{"interface": "$port_name", "direction": "tx"}},
			{Field: "SAI_PORT_STAT_IF_IN_FEC_CORRECTABLE_FRAMES", Metric: "sonic_switch_interface_fec_frames_total", Type: "counter", Help: "Total FEC frames", Labels: map[string]string{"interface": "$port_name", "type": "correctable"}},
			{Field: "SAI_PORT_STAT_IF_IN_FEC_NOT_CORRECTABLE_FRAMES", Metric: "sonic_switch_interface_fec_frames_total", Type: "counter", Help: "Total FEC frames", Labels: map[string]string{"interface": "$port_name", "type": "uncorrectable"}},
			{Field: "SAI_PORT_STAT_IF_IN_FEC_SYMBOL_ERRORS", Metric: "sonic_switch_interface_fec_frames_total", Type: "counter", Help: "Total FEC frames", Labels: map[string]string{"interface": "$port_name", "type": "symbol_errors"}},
			{Field: "SAI_PORT_STAT_IF_OUT_QLEN", Metric: "sonic_switch_interface_queue_length", Type: "gauge", Help: "Current output queue length", Labels: map[string]string{"interface": "$port_name"}},
			{Field: "SAI_PORT_STAT_PFC_0_RX_PKTS", Metric: "sonic_switch_interface_pfc_packets_total", Type: "counter", Help: "Total PFC packets", Labels: map[string]string{"interface": "$port_name", "direction": "rx", "priority": "0"}},
			{Field: "SAI_PORT_STAT_PFC_0_TX_PKTS", Metric: "sonic_switch_interface_pfc_packets_total", Type: "counter", Help: "Total PFC packets", Labels: map[string]string{"interface": "$port_name", "direction": "tx", "priority": "0"}},
			{Field: "SAI_PORT_STAT_ETHER_IN_PKTS_64_OCTETS", Metric: "sonic_switch_interface_packet_size_total", Type: "counter", Help: "Total packets by size bucket", Labels: map[string]string{"interface": "$port_name", "direction": "rx", "size": "64"}},
			{Field: "SAI_PORT_STAT_ETHER_OUT_PKTS_64_OCTETS", Metric: "sonic_switch_interface_packet_size_total", Type: "counter", Help: "Total packets by size bucket", Labels: map[string]string{"interface": "$port_name", "direction": "tx", "size": "64"}},
			{Field: "SAI_PORT_STAT_ETHER_STATS_UNDERSIZE_PKTS", Metric: "sonic_switch_interface_anomaly_packets_total", Type: "counter", Help: "Total anomalous packets", Labels: map[string]string{"interface": "$port_name", "type": "undersize"}},
			{Field: "SAI_PORT_STAT_ETHER_STATS_FRAGMENTS", Metric: "sonic_switch_interface_anomaly_packets_total", Type: "counter", Help: "Total anomalous packets", Labels: map[string]string{"interface": "$port_name", "type": "fragments"}},
			{Field: "SAI_PORT_STAT_ETHER_STATS_JABBERS", Metric: "sonic_switch_interface_anomaly_packets_total", Type: "counter", Help: "Total anomalous packets", Labels: map[string]string{"interface": "$port_name", "type": "jabbers"}},
			{Field: "SAI_PORT_STAT_IF_IN_UNKNOWN_PROTOS", Metric: "sonic_switch_interface_anomaly_packets_total", Type: "counter", Help: "Total anomalous packets", Labels: map[string]string{"interface": "$port_name", "type": "unknown_protos"}},
			{Field: "SAI_PORT_STAT_ETHER_RX_OVERSIZE_PKTS", Metric: "sonic_switch_interface_anomaly_packets_total", Type: "counter", Help: "Total anomalous packets", Labels: map[string]string{"interface": "$port_name", "type": "rx_oversize"}},
			{Field: "SAI_PORT_STAT_ETHER_TX_OVERSIZE_PKTS", Metric: "sonic_switch_interface_anomaly_packets_total", Type: "counter", Help: "Total anomalous packets", Labels: map[string]string{"interface": "$port_name", "type": "tx_oversize"}},
		},
	}

	collector := NewConfigCollector(mc, mapping)

	// Verify a representative subset of metrics
	expected := `
		# HELP sonic_switch_interface_bytes_total Total bytes transferred
		# TYPE sonic_switch_interface_bytes_total counter
		sonic_switch_interface_bytes_total{direction="rx",interface="Ethernet0"} 1.23456789e+08
		sonic_switch_interface_bytes_total{direction="tx",interface="Ethernet0"} 9.87654321e+08
		# HELP sonic_switch_interface_errors_total Total interface errors
		# TYPE sonic_switch_interface_errors_total counter
		sonic_switch_interface_errors_total{direction="rx",interface="Ethernet0"} 42
		sonic_switch_interface_errors_total{direction="tx",interface="Ethernet0"} 0
		# HELP sonic_switch_interface_discards_total Total interface discards
		# TYPE sonic_switch_interface_discards_total counter
		sonic_switch_interface_discards_total{direction="rx",interface="Ethernet0"} 7
		sonic_switch_interface_discards_total{direction="tx",interface="Ethernet0"} 3
		# HELP sonic_switch_interface_dropped_packets_total Total SAI-level dropped packets
		# TYPE sonic_switch_interface_dropped_packets_total counter
		sonic_switch_interface_dropped_packets_total{direction="rx",interface="Ethernet0"} 2
		sonic_switch_interface_dropped_packets_total{direction="tx",interface="Ethernet0"} 1
		# HELP sonic_switch_interface_fec_frames_total Total FEC frames
		# TYPE sonic_switch_interface_fec_frames_total counter
		sonic_switch_interface_fec_frames_total{interface="Ethernet0",type="correctable"} 1580
		sonic_switch_interface_fec_frames_total{interface="Ethernet0",type="symbol_errors"} 23
		sonic_switch_interface_fec_frames_total{interface="Ethernet0",type="uncorrectable"} 0
		# HELP sonic_switch_interface_queue_length Current output queue length
		# TYPE sonic_switch_interface_queue_length gauge
		sonic_switch_interface_queue_length{interface="Ethernet0"} 5
		# HELP sonic_switch_interface_packets_total Total packets transferred
		# TYPE sonic_switch_interface_packets_total counter
		sonic_switch_interface_packets_total{direction="rx",interface="Ethernet0",type="broadcast"} 50
		sonic_switch_interface_packets_total{direction="rx",interface="Ethernet0",type="multicast"} 500
		sonic_switch_interface_packets_total{direction="rx",interface="Ethernet0",type="non_unicast"} 550
		sonic_switch_interface_packets_total{direction="rx",interface="Ethernet0",type="unicast"} 100000
		sonic_switch_interface_packets_total{direction="tx",interface="Ethernet0",type="broadcast"} 20
		sonic_switch_interface_packets_total{direction="tx",interface="Ethernet0",type="multicast"} 300
		sonic_switch_interface_packets_total{direction="tx",interface="Ethernet0",type="non_unicast"} 320
		sonic_switch_interface_packets_total{direction="tx",interface="Ethernet0",type="unicast"} 200000
		# HELP sonic_switch_interface_pfc_packets_total Total PFC packets
		# TYPE sonic_switch_interface_pfc_packets_total counter
		sonic_switch_interface_pfc_packets_total{direction="rx",interface="Ethernet0",priority="0"} 100
		sonic_switch_interface_pfc_packets_total{direction="tx",interface="Ethernet0",priority="0"} 50
		# HELP sonic_switch_interface_packet_size_total Total packets by size bucket
		# TYPE sonic_switch_interface_packet_size_total counter
		sonic_switch_interface_packet_size_total{direction="rx",interface="Ethernet0",size="64"} 10000
		sonic_switch_interface_packet_size_total{direction="tx",interface="Ethernet0",size="64"} 8000
		# HELP sonic_switch_interface_anomaly_packets_total Total anomalous packets
		# TYPE sonic_switch_interface_anomaly_packets_total counter
		sonic_switch_interface_anomaly_packets_total{interface="Ethernet0",type="fragments"} 0
		sonic_switch_interface_anomaly_packets_total{interface="Ethernet0",type="jabbers"} 0
		sonic_switch_interface_anomaly_packets_total{interface="Ethernet0",type="rx_oversize"} 0
		sonic_switch_interface_anomaly_packets_total{interface="Ethernet0",type="tx_oversize"} 0
		sonic_switch_interface_anomaly_packets_total{interface="Ethernet0",type="undersize"} 0
		sonic_switch_interface_anomaly_packets_total{interface="Ethernet0",type="unknown_protos"} 0
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("ConfigCollector counters mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

func TestConfigCollectorMapTransform(t *testing.T) {
	mc := newMockConnector("STATE_DB")

	mc.mocks["STATE_DB"].ExpectKeys("PORT_TABLE|*").SetVal([]string{
		"PORT_TABLE|Ethernet0",
	})
	mc.mocks["STATE_DB"].ExpectHGetAll("PORT_TABLE|Ethernet0").SetVal(map[string]string{
		"oper_status": "up",
	})

	mapping := MetricMapping{
		RedisDB:      "STATE_DB",
		KeyPattern:   "PORT_TABLE|*",
		KeySeparator: "|",
		Fields: []FieldMapping{
			{
				Field:  "oper_status",
				Metric: "sonic_switch_interface_oper_state",
				Type:   "gauge",
				Help:   "Operational state of the interface",
				Labels: map[string]string{
					"interface": "$key_suffix",
				},
				Transform: &Transform{
					Map: map[string]float64{"up": 1, "down": 0},
				},
			},
		},
	}

	collector := NewConfigCollector(mc, mapping)
	expected := `
		# HELP sonic_switch_interface_oper_state Operational state of the interface
		# TYPE sonic_switch_interface_oper_state gauge
		sonic_switch_interface_oper_state{interface="Ethernet0"} 1
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("ConfigCollector map transform mismatch: %v", err)
	}
	mc.expectationsMet(t)
}

// --- Config Loading Tests ---

func TestDefaultConfigLoads(t *testing.T) {
	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() failed: %v", err)
	}
	if len(cfg.Metrics) == 0 {
		t.Fatal("DefaultConfig() returned empty metrics")
	}
	// Verify expected metric mappings exist
	metricNames := make(map[string]bool)
	for _, m := range cfg.Metrics {
		for _, f := range m.Fields {
			metricNames[f.Metric] = true
		}
	}
	for _, want := range []string{
		"sonic_switch_transceiver_dom_threshold",
		"sonic_switch_transceiver_info",
		"sonic_switch_interface_errors_total",
		"sonic_switch_interface_discards_total",
		"sonic_switch_interface_fec_frames_total",
		"sonic_switch_transceiver_dom_temperature_celsius",
		"sonic_switch_transceiver_dom_voltage_volts",
		"sonic_switch_transceiver_dom_rx_power_dbm",
		"sonic_switch_transceiver_dom_tx_bias_milliamps",
		"sonic_switch_transceiver_rxlos",
		"sonic_switch_transceiver_txfault",
		"sonic_switch_interface_neighbor_info",
		"sonic_switch_temperature_celsius",
		"sonic_switch_temperature_high_threshold_celsius",
		"sonic_switch_temperature_warning",
		"sonic_switch_interface_bytes_total",
		"sonic_switch_interface_packets_total",
		"sonic_switch_interface_dropped_packets_total",
		"sonic_switch_interface_queue_length",
		"sonic_switch_interface_pfc_packets_total",
		"sonic_switch_interface_packet_size_total",
		"sonic_switch_interface_anomaly_packets_total",
	} {
		if !metricNames[want] {
			t.Errorf("DefaultConfig missing expected metric %q", want)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     MetricsConfig
		wantErr bool
	}{
		{"valid", MetricsConfig{Metrics: []MetricMapping{{
			RedisDB: "STATE_DB", KeyPattern: "FOO|*",
			Fields: []FieldMapping{{Metric: "test_metric", Type: "gauge"}},
		}}}, false},
		{"missing redis_db", MetricsConfig{Metrics: []MetricMapping{{
			KeyPattern: "FOO|*",
			Fields:     []FieldMapping{{Metric: "test_metric", Type: "gauge"}},
		}}}, true},
		{"missing key_pattern", MetricsConfig{Metrics: []MetricMapping{{
			RedisDB: "STATE_DB",
			Fields:  []FieldMapping{{Metric: "test_metric", Type: "gauge"}},
		}}}, true},
		{"missing metric name", MetricsConfig{Metrics: []MetricMapping{{
			RedisDB: "STATE_DB", KeyPattern: "FOO|*",
			Fields: []FieldMapping{{Type: "gauge"}},
		}}}, true},
		{"invalid type", MetricsConfig{Metrics: []MetricMapping{{
			RedisDB: "STATE_DB", KeyPattern: "FOO|*",
			Fields: []FieldMapping{{Metric: "test_metric", Type: "histogram"}},
		}}}, true},
		{"field and field_pattern exclusive", MetricsConfig{Metrics: []MetricMapping{{
			RedisDB: "STATE_DB", KeyPattern: "FOO|*",
			Fields: []FieldMapping{{Metric: "test_metric", Type: "gauge", Field: "foo", FieldPattern: "*"}},
		}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- Health Endpoint Test ---

func TestHealthEndpointOK(t *testing.T) {
	mc := newMockConnector("CONFIG_DB")
	mc.mocks["CONFIG_DB"].ExpectPing().SetVal("PONG")

	srv := NewMetricsServer(":0", mc, nil, "")

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected 'ok', got %q", w.Body.String())
	}
	mc.expectationsMet(t)
}

func TestHealthEndpointRedisDown(t *testing.T) {
	mc := newMockConnector()

	srv := NewMetricsServer(":0", mc, nil, "")

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- Scrape Duration Test ---

func TestScrapeDurationMetric(t *testing.T) {
	mc := newMockConnector("CONFIG_DB")
	// DeviceCollector will read DEVICE_METADATA
	mc.mocks["CONFIG_DB"].ExpectHGetAll("DEVICE_METADATA|localhost").SetVal(map[string]string{
		"mac": "aa:bb:cc:dd:ee:ff",
	})
	// InterfaceCollector will list ports
	mc.mocks["CONFIG_DB"].ExpectKeys("PORT|*").SetVal([]string{})

	srv := NewMetricsServer(":0", mc, nil, "")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "sonic_scrape_duration_seconds") {
		t.Error("response missing sonic_scrape_duration_seconds metric")
	}
}

// --- Error Handling Test ---

func TestDeviceCollectorRedisDown(t *testing.T) {
	mc := newMockConnector()
	collector := NewDeviceCollector(mc, nil)

	reg := prometheus.NewRegistry()
	reg.MustRegister(collector)
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	found := false
	for _, mf := range metrics {
		if mf.GetName() == "sonic_switch_ready" {
			found = true
			if mf.GetMetric()[0].GetGauge().GetValue() != 0 {
				t.Errorf("expected sonic_switch_ready=0, got %v", mf.GetMetric()[0].GetGauge().GetValue())
			}
		}
	}
	if !found {
		t.Error("sonic_switch_ready metric not found")
	}
}

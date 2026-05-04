# Agent metrics

The switch agent exposes a Prometheus-compatible `/metrics` endpoint for monitoring switch health, interface state, and transceiver optics. Metrics are collected just-in-time from SONiC Redis on every Prometheus scrape — there is no background polling or caching.

## Endpoints

| Path | Description |
|------|-------------|
| `/metrics` | Prometheus metrics |
| `/healthz` | Health check — returns `200 OK` if Redis is reachable, `500` otherwise |

## Configuration

The agent accepts two flags for metrics:

| Flag | Default | Description |
|------|---------|-------------|
| `-metrics-port` | `9100` | HTTP port for the metrics server |
| `-metrics-config` | _(empty)_ | Path to a custom metrics mapping YAML. When empty, the built-in default config is used |

## Metric types

Metrics come from two sources:

### Built-in collectors

These require custom logic (cross-database joins, aggregate counting, error fallbacks) and are always registered.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `sonic_switch_info` | gauge | `mac`, `firmware`, `hwsku`, `asic`, `platform` | Device metadata, always 1. Firmware and ASIC fall back to `/etc/sonic/sonic_version.yml` when absent from Redis |
| `sonic_switch_ready` | gauge | — | 1 if the switch is ready, 0 otherwise |
| `sonic_switch_interface_oper_state` | gauge | `interface` | Operational state (1=up, 0=down) |
| `sonic_switch_interface_admin_state` | gauge | `interface` | Admin state (1=up, 0=down) |
| `sonic_switch_interfaces_total` | gauge | `operational_status` | Number of interfaces by status |
| `sonic_switch_ports_total` | gauge | — | Total physical ports |
| `sonic_scrape_duration_seconds` | gauge | — | Duration of the last metrics scrape |

### Config-driven collectors

These are defined in YAML and can be customized or extended by operators. The default config ships the following metrics:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `sonic_switch_transceiver_dom_temperature_celsius` | gauge | `interface` | Transceiver temperature |
| `sonic_switch_transceiver_dom_voltage_volts` | gauge | `interface` | Transceiver supply voltage |
| `sonic_switch_transceiver_dom_rx_power_dbm` | gauge | `interface`, `lane` | RX power per lane |
| `sonic_switch_transceiver_dom_tx_bias_milliamps` | gauge | `interface`, `lane` | TX bias current per lane |
| `sonic_switch_transceiver_dom_threshold` | gauge | `interface`, `sensor`, `level`, `direction` | DOM threshold values |
| `sonic_switch_transceiver_info` | gauge | `interface`, `type`, `vendor`, `model`, `serial` | Transceiver metadata, always 1 |
| `sonic_switch_transceiver_rxlos` | gauge | `interface`, `lane` | RX loss of signal per lane (1=loss, 0=ok) |
| `sonic_switch_transceiver_txfault` | gauge | `interface`, `lane` | TX fault per lane (1=fault, 0=ok) |
| `sonic_switch_interface_neighbor_info` | gauge | `interface`, `neighbor_mac`, `neighbor_name`, `neighbor_port` | LLDP neighbor metadata, always 1 |
| `sonic_switch_temperature_celsius` | gauge | `sensor` | Chassis temperature sensor reading |
| `sonic_switch_temperature_high_threshold_celsius` | gauge | `sensor` | Chassis temperature sensor high threshold |
| `sonic_switch_temperature_warning` | gauge | `sensor` | Chassis temperature warning status (1=warning, 0=ok) |
| `sonic_switch_interface_bytes_total` | counter | `interface`, `direction` | Bytes transferred |
| `sonic_switch_interface_packets_total` | counter | `interface`, `direction`, `type` | Packets by type (unicast, multicast, broadcast, non_unicast) |
| `sonic_switch_interface_errors_total` | counter | `interface`, `direction` | Interface error counters |
| `sonic_switch_interface_discards_total` | counter | `interface`, `direction` | Interface discard counters |
| `sonic_switch_interface_dropped_packets_total` | counter | `interface`, `direction` | SAI-level dropped packets |
| `sonic_switch_interface_fec_frames_total` | counter | `interface`, `type` | FEC frame counters (correctable, uncorrectable, symbol_errors) |
| `sonic_switch_interface_queue_length` | gauge | `interface` | Current output queue length |
| `sonic_switch_interface_pfc_packets_total` | counter | `interface`, `direction`, `priority` | PFC packets per priority (0-7) |
| `sonic_switch_interface_rx_packet_size_bytes` | histogram | `interface` | RX packet size distribution (buckets: 64, 127, 255, 511, 1023, 1518, 2047, 4095, 9216, 16383) |
| `sonic_switch_interface_tx_packet_size_bytes` | histogram | `interface` | TX packet size distribution (buckets: 64, 127, 255, 511, 1023, 1518, 2047, 4095, 9216, 16383) |
| `sonic_switch_interface_anomaly_packets_total` | counter | `interface`, `type` | Anomalous packets (undersize, oversize, fragments, jabbers, unknown_protos) |

## Metrics configuration schema

A custom config file replaces all config-driven metrics. The file is YAML with a single top-level key:

```yaml
metrics:
  - redis_db: ...
    key_pattern: ...
    fields:
      - metric: ...
        ...
```

### `metrics[]` — Metric mapping

Each entry maps a set of Redis keys to one or more Prometheus metrics.

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `redis_db` | yes | — | SONiC Redis database name (`CONFIG_DB`, `STATE_DB`, `COUNTERS_DB`, `APPL_DB`) |
| `key_pattern` | yes | — | Redis `KEYS` glob pattern (e.g. `TRANSCEIVER_INFO|*`) |
| `key_separator` | no | `\|` | Character separating the table prefix from the key suffix |
| `key_resolver` | no | — | Name of a Redis hash that maps logical names to key suffixes (e.g. `COUNTERS_PORT_NAME_MAP`) |
| `fields` | yes | — | List of field-to-metric mappings |

### `fields[]` — Field mapping

Each entry maps a Redis hash field (or set of fields) to a Prometheus metric.

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `field` | no | — | Specific Redis hash field name. Mutually exclusive with `field_pattern` |
| `field_pattern` | no | — | Set to `*` to iterate all hash fields. Mutually exclusive with `field` |
| `metric` | yes | — | Prometheus metric name |
| `type` | yes | — | `gauge`, `counter`, or `histogram` |
| `help` | no | — | Metric help string |
| `value` | no | — | Fixed metric value (ignores field value). Use for `_info` pattern metrics |
| `labels` | no | — | Map of label names to [value templates](#label-value-templates) |
| `transform` | no | — | [Value transformation](#transforms) |

When neither `field` nor `field_pattern` is set, the metric is emitted once per key using `value` or label data from the hash.

### Label value templates

Label values are strings that can reference dynamic data using `$` prefixes:

| Template | Resolves to |
|----------|-------------|
| `$key_suffix` | Part of the Redis key after the separator (e.g. `Ethernet0` from `TRANSCEIVER_INFO\|Ethernet0`) |
| `$port_name` | Resolved name from `key_resolver` (e.g. `Ethernet0` resolved via `COUNTERS_PORT_NAME_MAP`) |
| `$field_name` | The Redis hash field name (useful with `field_pattern: "*"`) |
| `$<hash_field>` | Value of a hash field (e.g. `$vendor_name` reads the `vendor_name` field) |
| _(literal)_ | Any string without a `$` prefix is used as-is |

### Transforms

Transforms modify how the metric value is derived. At most one transform should be set per field mapping.

#### `map`

Maps string field values to floats. Unmapped values are silently skipped.

```yaml
transform:
  map:
    up: 1
    down: 0
```

#### `regex_capture`

Matches field names against a Go regex with [named capture groups](https://pkg.go.dev/regexp/syntax). Non-matching fields are skipped. Capture group names become additional Prometheus labels. Requires `field_pattern: "*"`.

```yaml
field_pattern: "*"
metric: sonic_switch_transceiver_dom_rx_power_dbm
labels:
  interface: "$key_suffix"
transform:
  regex_capture:
    pattern: "^rx(?P<lane>\\d+)power$"
```

This matches `rx1power`, `rx2power`, etc. and produces a `lane` label with the captured digit.

`regex_capture` can be combined with `map` to filter field names by regex while also converting string values. For example, to expose per-lane boolean fields as numeric gauges:

```yaml
field_pattern: "*"
metric: sonic_switch_transceiver_rxlos
labels:
  interface: "$key_suffix"
transform:
  regex_capture:
    pattern: "^rxlos(?P<lane>\\d+)$"
  map:
    "True": 1
    "False": 0
```

#### `parse_threshold_field`

Parses SONiC DOM threshold field names (e.g. `temphighalarm`) into three additional labels: `sensor`, `level`, and `direction`. Requires `field_pattern: "*"`.

```yaml
transform:
  parse_threshold_field: true
```

| Field name | sensor | level | direction |
|------------|--------|-------|-----------|
| `temphighalarm` | temperature | alarm | high |
| `vcclowwarning` | voltage | warning | low |
| `rxpowerhighwarning` | rx_power | warning | high |
| `txbiaslowalarm` | tx_bias | alarm | low |
| `txpowerhighalarm` | tx_power | alarm | high |

#### `dom_flag_severity`

Computes a severity rollup from all DOM flag fields in the hash. Each field is parsed as a threshold field name; if its value is `"true"`, it contributes to the severity. Returns the highest severity found: `0` (ok), `1` (warning), or `2` (alarm). Note: this transform is available but not included in the default config because the `TRANSCEIVER_DOM_FLAG` table is not present on all platforms.

```yaml
transform:
  dom_flag_severity: true
```

#### `histogram`

Maps multiple Redis hash fields to a single Prometheus histogram. Each entry in `buckets` maps an upper bound (float64) to a Redis hash field name. The transform reads each field, parses the count as an unsigned integer, and accumulates cumulative bucket counts. The resulting histogram has `sum=0` because SAI counters don't provide total bytes — but bucket-based percentile queries and heatmap visualizations still work. Requires `type: "histogram"`.

```yaml
- metric: sonic_switch_interface_rx_packet_size_bytes
  type: histogram
  help: "RX packet size distribution"
  labels:
    interface: "$port_name"
  transform:
    histogram:
      buckets:
        64: SAI_PORT_STAT_ETHER_IN_PKTS_64_OCTETS
        127: SAI_PORT_STAT_ETHER_IN_PKTS_65_TO_127_OCTETS
        255: SAI_PORT_STAT_ETHER_IN_PKTS_128_TO_255_OCTETS
        511: SAI_PORT_STAT_ETHER_IN_PKTS_256_TO_511_OCTETS
        1023: SAI_PORT_STAT_ETHER_IN_PKTS_512_TO_1023_OCTETS
        1518: SAI_PORT_STAT_ETHER_IN_PKTS_1024_TO_1518_OCTETS
```

This emits `_bucket`, `_count`, and `_sum` series automatically — Prometheus handles the histogram suffixes.

## Examples

### Adding a new counter from COUNTERS_DB

```yaml
metrics:
  - redis_db: COUNTERS_DB
    key_pattern: "COUNTERS:*"
    key_separator: ":"
    key_resolver: COUNTERS_PORT_NAME_MAP
    fields:
      - field: SAI_PORT_STAT_IF_IN_UCAST_PKTS
        metric: sonic_switch_interface_unicast_packets_total
        type: counter
        help: "Total unicast packets received"
        labels:
          interface: "$port_name"
          direction: "rx"
```

### Exposing a string field as an enum gauge

```yaml
metrics:
  - redis_db: STATE_DB
    key_pattern: "PORT_TABLE|*"
    key_separator: "|"
    fields:
      - field: oper_status
        metric: sonic_switch_interface_oper_state
        type: gauge
        help: "Operational state of the interface"
        labels:
          interface: "$key_suffix"
        transform:
          map:
            up: 1
            down: 0
```

### Metadata as labels (info pattern)

```yaml
metrics:
  - redis_db: STATE_DB
    key_pattern: "TRANSCEIVER_INFO|*"
    key_separator: "|"
    fields:
      - metric: sonic_switch_transceiver_info
        type: gauge
        help: "Transceiver metadata"
        value: 1
        labels:
          interface: "$key_suffix"
          vendor: "$manufacturer"
          serial: "$serial"
```

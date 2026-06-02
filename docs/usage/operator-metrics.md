# Operator Metrics

The sonic-operator exposes standard [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/metrics) metrics on its metrics server. These provide insight into the reconciliation performance and health of the operator itself.

## Endpoint

The metrics server is configured with `--metrics-bind-address`. By default it is disabled (`0`). To enable:

```
--metrics-bind-address=:8443                   # HTTPS (default when non-zero)
--metrics-bind-address=:8080 --metrics-secure=false  # HTTP
```

Metrics are served at `/metrics` on the configured port.

## Available metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `controller_runtime_reconcile_total` | counter | `controller`, `result` | Total reconciliations per controller |
| `controller_runtime_reconcile_errors_total` | counter | `controller` | Reconciliation errors per controller |
| `controller_runtime_terminal_reconcile_errors_total` | counter | `controller` | Terminal (non-retryable) errors per controller |
| `controller_runtime_reconcile_panics_total` | counter | `controller` | Reconciliation panics per controller |
| `controller_runtime_reconcile_time_seconds` | histogram | `controller` | Reconciliation duration per controller |
| `controller_runtime_active_workers` | gauge | `controller` | Currently active workers per controller |
| `controller_runtime_max_concurrent_reconciles` | gauge | `controller` | Maximum concurrent reconciles per controller |

The `controller` label identifies which controller produced the metric (e.g. `switch`, `switchinterface`).

## Scrape configuration

The operator metrics server also hosts the [Metrics Discovery](/usage/service-discovery) endpoint at `/switch-sd`. You can scrape both the operator and the discovered switches from the same Prometheus job configuration:

```yaml
scrape_configs:
  # Operator itself
  - job_name: sonic-operator
    static_configs:
      - targets: ["sonic-operator.sonic-operator-system:8080"]

  # Switches (auto-discovered)
  - job_name: sonic-switches
    http_sd_configs:
      - url: http://sonic-operator.sonic-operator-system:8080/switch-sd
    relabel_configs:
      - source_labels: [__meta_sonic_switch_name]
        target_label: switch
```

## Useful queries

Reconciliation rate per controller:

```promql
rate(controller_runtime_reconcile_total[5m])
```

Error ratio:

```promql
rate(controller_runtime_reconcile_errors_total[5m])
/ rate(controller_runtime_reconcile_total[5m])
```

p99 reconciliation latency:

```promql
histogram_quantile(0.99, rate(controller_runtime_reconcile_time_seconds_bucket[5m]))
```

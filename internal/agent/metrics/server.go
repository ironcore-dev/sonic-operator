// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServer creates an HTTP server that serves Prometheus metrics on /metrics
// and a health check on /healthz. All metrics are collected just-in-time from Redis.
//
// versionInfo is optional: when non-nil it provides fallback device metadata.
// configPath is optional: when empty, the embedded default config is used.
// When set, the config is loaded from that file path.
func NewMetricsServer(addr string, connector RedisConnector, versionInfo VersionInfoFunc, configPath string) *http.Server {
	registry := prometheus.NewRegistry()

	// Register built-in collectors (require custom logic not expressible in config)
	registry.MustRegister(
		NewDeviceCollector(connector, versionInfo),
		NewInterfaceCollector(connector),
	)

	// Load and register config-driven collectors
	var cfg *MetricsConfig
	var err error
	if configPath != "" {
		cfg, err = LoadConfig(configPath)
	} else {
		cfg, err = DefaultConfig()
	}
	if err != nil {
		// Log but don't crash — built-in collectors still work
		fmt.Printf("WARNING: failed to load metrics config: %v\n", err)
	} else {
		for _, mapping := range cfg.Metrics {
			registry.MustRegister(NewConfigCollector(connector, mapping))
		}
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		client, err := connector.Connect("CONFIG_DB")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "redis unhealthy: %v", err)
			return
		}
		if err := client.Ping(r.Context()).Err(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "redis ping failed: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"net/http"
	"time"

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

	// Scrape duration gauge — updated after each /metrics response is written,
	// so it reports the duration of the previous scrape (not the current one).
	// This is consistent with how node_exporter and similar exporters work.
	scrapeDuration := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sonic_scrape_duration_seconds",
		Help: "Duration of the last metrics scrape in seconds",
	})
	registry.MustRegister(scrapeDuration)

	mux := http.NewServeMux()
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	mux.Handle("GET /metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler.ServeHTTP(w, r)
		scrapeDuration.Set(time.Since(start).Seconds())
	}))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		client, err := connector.Connect("CONFIG_DB")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "redis unhealthy: %v", err)
			return
		}
		if err := client.Ping(r.Context()).Err(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "redis ping failed: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

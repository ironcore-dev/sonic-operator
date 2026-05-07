// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	metricsPort = 9100
	listTimeout = 10 * time.Second
	cacheTTL    = 10 * time.Second
)

// TargetGroup is a Prometheus HTTP SD target group.
type TargetGroup struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// NewHandler returns an http.Handler that serves Prometheus HTTP SD target
// groups for all Ready switches. Responses are cached for 10s to bound
// Kubernetes API load when many Prometheus instances scrape concurrently.
func NewHandler(c client.Reader) http.Handler {
	return &handler{client: c}
}

type handler struct {
	client client.Reader

	mu       sync.Mutex
	cached   []byte
	cachedAt time.Time
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	if time.Since(h.cachedAt) < cacheTTL && h.cached != nil {
		data := h.cached
		h.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
		return
	}
	h.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), listTimeout)
	defer cancel()

	var switches networkingv1alpha1.SwitchList
	if err := h.client.List(ctx, &switches); err != nil {
		log.Printf("switch-sd: failed to list switches: %v", err)
		http.Error(w, "failed to list switches", http.StatusInternalServerError)
		return
	}

	groups := buildTargetGroups(switches.Items)

	data, err := json.Marshal(groups)
	if err != nil {
		log.Printf("switch-sd: failed to encode response: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.cached = data
	h.cachedAt = time.Now()
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func buildTargetGroups(switches []networkingv1alpha1.Switch) []TargetGroup {
	groups := make([]TargetGroup, 0, len(switches))
	for _, sw := range switches {
		if sw.Status.State != networkingv1alpha1.SwitchStateReady {
			continue
		}
		if sw.Spec.Management.Host == "" {
			continue
		}

		labels := map[string]string{
			"__meta_sonic_switch_name": sw.Name,
		}
		if sw.Status.MACAddress != "" {
			labels["__meta_sonic_switch_mac"] = sw.Status.MACAddress
		}
		if sw.Status.SKU != "" {
			labels["__meta_sonic_switch_sku"] = sw.Status.SKU
		}
		if sw.Status.FirmwareVersion != "" {
			labels["__meta_sonic_switch_firmware"] = sw.Status.FirmwareVersion
		}

		groups = append(groups, TargetGroup{
			Targets: []string{net.JoinHostPort(sw.Spec.Management.Host, fmt.Sprintf("%d", metricsPort))},
			Labels:  labels,
		})
	}
	return groups
}

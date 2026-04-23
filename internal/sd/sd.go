// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const metricsPort = 9100

// TargetGroup is a Prometheus HTTP SD target group.
type TargetGroup struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

type handler struct {
	client client.Reader
}

// Register adds the /switch-sd HTTP service discovery endpoint to the mux.
func Register(mux *http.ServeMux, c client.Reader) {
	mux.Handle("GET /switch-sd", &handler{client: c})
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var switches networkingv1alpha1.SwitchList
	if err := h.client.List(r.Context(), &switches); err != nil {
		log.Printf("switch-sd: failed to list switches: %v", err)
		http.Error(w, "failed to list switches", http.StatusInternalServerError)
		return
	}

	groups := buildTargetGroups(switches.Items)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(groups); err != nil {
		log.Printf("switch-sd: failed to encode response: %v", err)
	}
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
			Targets: []string{fmt.Sprintf("%s:%d", sw.Spec.Management.Host, metricsPort)},
			Labels:  labels,
		})
	}
	return groups
}

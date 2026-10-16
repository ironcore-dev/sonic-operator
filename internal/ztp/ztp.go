// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ztp

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
)

type configMapHandler struct {
	reader client.Reader
}

type generatedHandler struct {
	reader                    client.Reader
	controlKubeconfigFilePath string
}

// GeneratedOptions configures declarative, Switch-backed ZTP rendering.
type GeneratedOptions struct {
	// ControlKubeconfigFile is an optional kubeconfig mounted into the operator
	// pod. It is injected only into generated containers that
	// explicitly opt in.
	ControlKubeconfigFile string
}

// RegisterConfigMap serves the script referenced by the Switch whose ZTP
// source address matches the requesting client. It deliberately has no
// template fallback: configmap mode must fail closed.
func RegisterConfigMap(mux *http.ServeMux, reader client.Reader) {
	mux.Handle("GET /ztp", &configMapHandler{reader: reader})
}

// RegisterGenerated renders a complete ZTP script from the matching Switch
// object. Unlike configmap mode, no user-provided ZTP script is read or
// modified.
func RegisterGenerated(mux *http.ServeMux, reader client.Reader, options GeneratedOptions) {
	mux.Handle("GET /ztp", &generatedHandler{
		reader:                    reader,
		controlKubeconfigFilePath: options.ControlKubeconfigFile,
	})
}

func (h *configMapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	matched, err := matchingSwitch(r, h.reader)
	if err != nil {
		handleErr(w, err)
		return
	}
	if matched.Spec.ZTP.ScriptRef == nil {
		handleErr(w, fmt.Errorf("switch %q has no ztp.scriptRef for configmap mode", matched.Name))
		return
	}

	ref := matched.Spec.ZTP.ScriptRef
	configMap := &corev1.ConfigMap{}
	if err := h.reader.Get(r.Context(), types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, configMap); err != nil {
		handleErr(w, fmt.Errorf("getting ZTP ConfigMap %s/%s: %w", ref.Namespace, ref.Name, err))
		return
	}
	script, ok := configMap.Data[ref.Key]
	if !ok {
		handleErr(w, fmt.Errorf("ZTP ConfigMap %s/%s has no key %q", ref.Namespace, ref.Name, ref.Key))
		return
	}

	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	if _, err := fmt.Fprint(w, script); err != nil {
		slog.Error("failed to write ZTP ConfigMap script", "switch", matched.Name, "err", err)
	}
}

func (h *generatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	matched, err := matchingSwitch(r, h.reader)
	if err != nil {
		handleErr(w, err)
		return
	}

	script, err := renderGeneratedScript(matched, h.controlKubeconfigFilePath)
	if err != nil {
		handleErr(w, fmt.Errorf("rendering generated ZTP for switch %q: %w", matched.Name, err))
		return
	}

	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	if _, err := fmt.Fprint(w, script); err != nil {
		slog.Error("failed to write generated ZTP script", "switch", matched.Name, "err", err)
	}
}

func matchingSwitch(r *http.Request, reader client.Reader) (*networkingv1alpha1.Switch, error) {
	ap, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}

	switches := &networkingv1alpha1.SwitchList{}
	if err := reader.List(r.Context(), switches); err != nil {
		return nil, fmt.Errorf("listing switches for ZTP: %w", err)
	}

	var matched *networkingv1alpha1.Switch
	for i := range switches.Items {
		switchConfig := switches.Items[i].Spec.ZTP
		if switchConfig == nil {
			continue
		}
		configuredAddr, err := netip.ParseAddr(switchConfig.SourceAddress)
		if err != nil {
			return nil, fmt.Errorf("switch %q has invalid ZTP source address %q: %w", switches.Items[i].Name, switchConfig.SourceAddress, err)
		}
		if configuredAddr != ap.Addr() {
			continue
		}
		if matched != nil {
			return nil, fmt.Errorf("multiple switches use ZTP source address %q", ap.Addr())
		}
		matched = &switches.Items[i]
	}
	if matched == nil {
		return nil, fmt.Errorf("no switch configured for ZTP source address %q", ap.Addr())
	}
	return matched, nil
}

func handleErr(w http.ResponseWriter, e error) {
	w.WriteHeader(http.StatusInternalServerError)
	_, err := fmt.Fprint(w, e.Error())
	if err != nil {
		slog.Error("failed to write back previous error", "e", e, "err", err)
	}
}

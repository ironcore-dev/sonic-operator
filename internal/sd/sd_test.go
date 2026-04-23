// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = networkingv1alpha1.AddToScheme(s)
	return s
}

func get(t *testing.T, handler http.Handler) []TargetGroup {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/switch-sd", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var groups []TargetGroup
	if err := json.NewDecoder(rec.Body).Decode(&groups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return groups
}

func TestNoSwitches(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	mux := http.NewServeMux()
	Register(mux, c)

	groups := get(t, mux)
	if len(groups) != 0 {
		t.Errorf("expected 0 target groups, got %d", len(groups))
	}
}

func TestReadySwitch(t *testing.T) {
	sw := &networkingv1alpha1.Switch{
		ObjectMeta: metav1.ObjectMeta{Name: "leaf-1"},
		Spec: networkingv1alpha1.SwitchSpec{
			MacAddress: "aa:bb:cc:dd:ee:ff",
			Management: networkingv1alpha1.Management{
				Host: "10.0.0.1",
				Port: "50051",
			},
		},
		Status: networkingv1alpha1.SwitchStatus{
			State:           networkingv1alpha1.SwitchStateReady,
			MACAddress:      "aa:bb:cc:dd:ee:ff",
			SKU:             "Accton-AS7726-32X",
			FirmwareVersion: "4.2.0",
		},
	}
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(sw).WithStatusSubresource(sw).Build()
	mux := http.NewServeMux()
	Register(mux, c)

	groups := get(t, mux)
	if len(groups) != 1 {
		t.Fatalf("expected 1 target group, got %d", len(groups))
	}
	g := groups[0]
	if len(g.Targets) != 1 || g.Targets[0] != "10.0.0.1:9100" {
		t.Errorf("unexpected targets: %v", g.Targets)
	}
	if g.Labels["__meta_sonic_switch_name"] != "leaf-1" {
		t.Errorf("unexpected name label: %s", g.Labels["__meta_sonic_switch_name"])
	}
	if g.Labels["__meta_sonic_switch_mac"] != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("unexpected mac label: %s", g.Labels["__meta_sonic_switch_mac"])
	}
	if g.Labels["__meta_sonic_switch_sku"] != "Accton-AS7726-32X" {
		t.Errorf("unexpected sku label: %s", g.Labels["__meta_sonic_switch_sku"])
	}
	if g.Labels["__meta_sonic_switch_firmware"] != "4.2.0" {
		t.Errorf("unexpected firmware label: %s", g.Labels["__meta_sonic_switch_firmware"])
	}
}

func TestMixedStates(t *testing.T) {
	switches := []networkingv1alpha1.Switch{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "leaf-1"},
			Spec: networkingv1alpha1.SwitchSpec{
				MacAddress: "aa:bb:cc:dd:ee:01",
				Management: networkingv1alpha1.Management{Host: "10.0.0.1", Port: "50051"},
			},
			Status: networkingv1alpha1.SwitchStatus{State: networkingv1alpha1.SwitchStateReady},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "leaf-2"},
			Spec: networkingv1alpha1.SwitchSpec{
				MacAddress: "aa:bb:cc:dd:ee:02",
				Management: networkingv1alpha1.Management{Host: "10.0.0.2", Port: "50051"},
			},
			Status: networkingv1alpha1.SwitchStatus{State: networkingv1alpha1.SwitchStatePending},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "leaf-3"},
			Spec: networkingv1alpha1.SwitchSpec{
				MacAddress: "aa:bb:cc:dd:ee:03",
				Management: networkingv1alpha1.Management{Host: "10.0.0.3", Port: "50051"},
			},
			Status: networkingv1alpha1.SwitchStatus{State: networkingv1alpha1.SwitchStateFailed},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "spine-1"},
			Spec: networkingv1alpha1.SwitchSpec{
				MacAddress: "aa:bb:cc:dd:ee:04",
				Management: networkingv1alpha1.Management{Host: "10.0.0.4", Port: "50051"},
			},
			Status: networkingv1alpha1.SwitchStatus{State: networkingv1alpha1.SwitchStateReady},
		},
	}

	objs := make([]networkingv1alpha1.Switch, len(switches))
	copy(objs, switches)
	builder := fake.NewClientBuilder().WithScheme(newScheme())
	for i := range objs {
		builder = builder.WithObjects(&objs[i]).WithStatusSubresource(&objs[i])
	}
	c := builder.Build()

	mux := http.NewServeMux()
	Register(mux, c)

	groups := get(t, mux)
	if len(groups) != 2 {
		t.Fatalf("expected 2 target groups (only Ready), got %d", len(groups))
	}

	targets := map[string]bool{}
	for _, g := range groups {
		for _, tgt := range g.Targets {
			targets[tgt] = true
		}
	}
	if !targets["10.0.0.1:9100"] {
		t.Error("expected leaf-1 in targets")
	}
	if !targets["10.0.0.4:9100"] {
		t.Error("expected spine-1 in targets")
	}
	if targets["10.0.0.2:9100"] {
		t.Error("pending switch should not be in targets")
	}
	if targets["10.0.0.3:9100"] {
		t.Error("failed switch should not be in targets")
	}
}

func TestEmptyHostSkipped(t *testing.T) {
	sw := &networkingv1alpha1.Switch{
		ObjectMeta: metav1.ObjectMeta{Name: "no-host"},
		Spec: networkingv1alpha1.SwitchSpec{
			MacAddress: "aa:bb:cc:dd:ee:ff",
			Management: networkingv1alpha1.Management{Host: "", Port: "50051"},
		},
		Status: networkingv1alpha1.SwitchStatus{State: networkingv1alpha1.SwitchStateReady},
	}
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(sw).WithStatusSubresource(sw).Build()
	mux := http.NewServeMux()
	Register(mux, c)

	groups := get(t, mux)
	if len(groups) != 0 {
		t.Errorf("expected 0 target groups for empty host, got %d", len(groups))
	}
}

// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(networkingv1alpha1.AddToScheme(s))
	return s
}

func TestHandler_NoSwitches(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	h := NewHandler(c)

	req := httptest.NewRequest(http.MethodGet, "/switch-sd", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var groups []TargetGroup
	if err := json.NewDecoder(rec.Body).Decode(&groups); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(groups))
	}
}

func TestHandler_ReadySwitchReturned(t *testing.T) {
	sw := &networkingv1alpha1.Switch{
		ObjectMeta: metav1.ObjectMeta{Name: "leaf1", Namespace: "default"},
		Spec: networkingv1alpha1.SwitchSpec{
			Management: networkingv1alpha1.Management{Host: "10.0.0.1"},
		},
		Status: networkingv1alpha1.SwitchStatus{
			State:           networkingv1alpha1.SwitchStateReady,
			MACAddress:      "aa:bb:cc:dd:ee:ff",
			SKU:             "AS7726-32X",
			FirmwareVersion: "4.2.0",
		},
	}

	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(sw).WithStatusSubresource(sw).Build()
	// Set status after creation since fake client doesn't persist status on create.
	sw.Status = networkingv1alpha1.SwitchStatus{
		State:           networkingv1alpha1.SwitchStateReady,
		MACAddress:      "aa:bb:cc:dd:ee:ff",
		SKU:             "AS7726-32X",
		FirmwareVersion: "4.2.0",
	}
	if err := c.Status().Update(context.Background(), sw); err != nil {
		t.Fatalf("status update: %v", err)
	}

	h := NewHandler(c)
	req := httptest.NewRequest(http.MethodGet, "/switch-sd", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var groups []TargetGroup
	if err := json.NewDecoder(rec.Body).Decode(&groups); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Targets[0] != "10.0.0.1:9100" {
		t.Errorf("expected target 10.0.0.1:9100, got %s", groups[0].Targets[0])
	}
	if groups[0].Labels["__meta_sonic_switch_name"] != "leaf1" {
		t.Errorf("expected label leaf1, got %s", groups[0].Labels["__meta_sonic_switch_name"])
	}
	if groups[0].Labels["__meta_sonic_switch_mac"] != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("expected mac label, got %s", groups[0].Labels["__meta_sonic_switch_mac"])
	}
}

func TestHandler_PendingSwitchExcluded(t *testing.T) {
	sw := &networkingv1alpha1.Switch{
		ObjectMeta: metav1.ObjectMeta{Name: "leaf2", Namespace: "default"},
		Spec: networkingv1alpha1.SwitchSpec{
			Management: networkingv1alpha1.Management{Host: "10.0.0.2"},
		},
		Status: networkingv1alpha1.SwitchStatus{
			State: networkingv1alpha1.SwitchStatePending,
		},
	}

	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(sw).WithStatusSubresource(sw).Build()
	sw.Status = networkingv1alpha1.SwitchStatus{State: networkingv1alpha1.SwitchStatePending}
	if err := c.Status().Update(context.Background(), sw); err != nil {
		t.Fatalf("status update: %v", err)
	}

	h := NewHandler(c)
	req := httptest.NewRequest(http.MethodGet, "/switch-sd", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var groups []TargetGroup
	if err := json.NewDecoder(rec.Body).Decode(&groups); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups for non-ready switch, got %d", len(groups))
	}
}

func TestHandler_IPv6Target(t *testing.T) {
	sw := &networkingv1alpha1.Switch{
		ObjectMeta: metav1.ObjectMeta{Name: "spine1", Namespace: "default"},
		Spec: networkingv1alpha1.SwitchSpec{
			Management: networkingv1alpha1.Management{Host: "2001:db8::1"},
		},
		Status: networkingv1alpha1.SwitchStatus{
			State: networkingv1alpha1.SwitchStateReady,
		},
	}

	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(sw).WithStatusSubresource(sw).Build()
	sw.Status = networkingv1alpha1.SwitchStatus{State: networkingv1alpha1.SwitchStateReady}
	if err := c.Status().Update(context.Background(), sw); err != nil {
		t.Fatalf("status update: %v", err)
	}

	h := NewHandler(c)
	req := httptest.NewRequest(http.MethodGet, "/switch-sd", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var groups []TargetGroup
	if err := json.NewDecoder(rec.Body).Decode(&groups); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Targets[0] != "[2001:db8::1]:9100" {
		t.Errorf("expected target [2001:db8::1]:9100, got %s", groups[0].Targets[0])
	}
}

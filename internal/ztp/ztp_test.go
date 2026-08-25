// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ztp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
)

func TestConfigMapHandlerServesMatchingSwitchScript(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&networkingv1alpha1.Switch{
			ObjectMeta: metav1.ObjectMeta{Name: "leaf-01"},
			Spec: networkingv1alpha1.SwitchSpec{ZTP: &networkingv1alpha1.ZTP{
				SourceAddress: "192.0.2.10",
				ScriptRef:     networkingv1alpha1.ZTPConfigMapReference{Namespace: "provisioning", Name: "leaf-01-ztp", Key: "ztp.sh"},
			}},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: "provisioning", Name: "leaf-01-ztp"},
			Data:       map[string]string{"ztp.sh": "#!/bin/bash\necho configured\n"},
		},
	).Build()

	mux := http.NewServeMux()
	RegisterConfigMap(mux, c)
	req := httptest.NewRequest(http.MethodGet, "http://provisioning.example/ztp", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got, want := response.Body.String(), "#!/bin/bash\necho configured\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Content-Type"), "text/x-shellscript; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func TestConfigMapHandlerDoesNotFallBackForUnknownSwitch(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterConfigMap(mux, fake.NewClientBuilder().WithScheme(scheme).Build())
	req := httptest.NewRequest(http.MethodGet, "http://provisioning.example/ztp", nil)
	req.RemoteAddr = "192.0.2.99:1234"
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := response.Body.String(); got == "" {
		t.Error("expected an error body for an unknown switch")
	}
}

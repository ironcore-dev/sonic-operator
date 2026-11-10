// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
)

func TestDeviceResourceLifecycleWithoutAgent(t *testing.T) {
	tests := []struct {
		name          string
		object        client.Object
		newReconciler func(client.Client) reconcile.Reconciler
	}{
		{
			name: "switch",
			object: &networkingv1alpha1.Switch{
				ObjectMeta: metav1.ObjectMeta{Name: "leaf-1"},
				Spec: networkingv1alpha1.SwitchSpec{
					Management: networkingv1alpha1.Management{Host: "192.0.2.10", Port: "57400"},
				},
				Status: networkingv1alpha1.SwitchStatus{
					MACAddress:      "aa:bb:cc:dd:ee:ff",
					FirmwareVersion: "existing-version",
					SKU:             "existing-sku",
					Ports:           []networkingv1alpha1.PortStatus{{Name: "Ethernet0"}},
				},
			},
			newReconciler: func(c client.Client) reconcile.Reconciler {
				return &SwitchReconciler{Client: c, Scheme: c.Scheme()}
			},
		},
		{
			name: "interface",
			object: &networkingv1alpha1.SwitchInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "leaf-1-ethernet0"},
				Spec: networkingv1alpha1.SwitchInterfaceSpec{
					Handle: "Ethernet0", NativeName: "Ethernet0",
					SwitchRef:  &corev1.LocalObjectReference{Name: "leaf-1"},
					AdminState: networkingv1alpha1.AdminStateDown,
				},
				Status: networkingv1alpha1.SwitchInterfaceStatus{
					State:            networkingv1alpha1.SwitchInterfaceStateReady,
					AdminState:       networkingv1alpha1.AdminStateUp,
					OperationalState: networkingv1alpha1.OperationStateUp,
					Neighbor:         networkingv1alpha1.Neighbor{SystemName: "existing-neighbor"},
				},
			},
			newReconciler: func(c client.Client) reconcile.Reconciler {
				return &SwitchInterfaceReconciler{Client: c, Scheme: c.Scheme()}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(tt.object).WithObjects(tt.object).Build()
			r := tt.newReconciler(c)
			ctx := context.Background()
			req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(tt.object)}
			want := tt.object.DeepCopyObject().(client.Object)
			switch obj := want.(type) {
			case *networkingv1alpha1.Switch:
				obj.Status.State = networkingv1alpha1.SwitchStateReady
			case *networkingv1alpha1.SwitchInterface:
				obj.Status.State = networkingv1alpha1.SwitchInterfaceStatePending
			}

			// Reconcile beyond initial finalizer/status setup: no polling or device discovery.
			for range 3 {
				result, err := r.Reconcile(ctx, req)
				if err != nil || result != (ctrl.Result{}) {
					t.Fatalf("Reconcile() = %v, %v; want no requeue or error", result, err)
				}
			}
			got := tt.object.DeepCopyObject().(client.Object)
			if err := c.Get(ctx, req.NamespacedName, got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.GetFinalizers(), []string{networkingv1alpha1.SwitchFinalizer}) {
				t.Fatalf("finalizers = %v", got.GetFinalizers())
			}
			// Ignore metadata changes; spec and observed status must be preserved.
			switch obj := got.(type) {
			case *networkingv1alpha1.Switch:
				obj.ObjectMeta = want.(*networkingv1alpha1.Switch).ObjectMeta
			case *networkingv1alpha1.SwitchInterface:
				obj.ObjectMeta = want.(*networkingv1alpha1.SwitchInterface).ObjectMeta
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("resource changed beyond readiness: got %#v, want %#v", got, want)
			}
			interfaces := &networkingv1alpha1.SwitchInterfaceList{}
			if err := c.List(ctx, interfaces); err != nil {
				t.Fatal(err)
			}
			wantInterfaces := 0
			if tt.name == "interface" {
				wantInterfaces = 1
			}
			if len(interfaces.Items) != wantInterfaces {
				t.Fatalf("unexpected interface discovery: %d interfaces", len(interfaces.Items))
			}

			if err := c.Delete(ctx, tt.object); err != nil {
				t.Fatal(err)
			}
			if _, err := r.Reconcile(ctx, req); err != nil {
				t.Fatal(err)
			}
			if err := c.Get(ctx, req.NamespacedName, got); !apierrors.IsNotFound(err) {
				t.Fatalf("resource still exists after finalizer cleanup: %v", err)
			}
			if _, err := r.Reconcile(ctx, req); err != nil {
				t.Fatalf("reconciling deleted resource: %v", err)
			}
		})
	}
}

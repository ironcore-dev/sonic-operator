// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/controller-utils/clientutils"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
)

// SwitchInterfaceReconciler reconciles a SwitchInterface object
type SwitchInterfaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=switchinterfaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=switchinterfaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=switchinterfaces/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SwitchInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	i := &networkingv1alpha1.SwitchInterface{}
	if err := r.Get(ctx, req.NamespacedName, i); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconileExists(ctx, log, i)
}

func (r *SwitchInterfaceReconciler) reconileExists(ctx context.Context, log logr.Logger, i *networkingv1alpha1.SwitchInterface) (ctrl.Result, error) {
	if !i.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, i)
	}
	return r.reconcile(ctx, log, i)
}

func (r *SwitchInterfaceReconciler) delete(ctx context.Context, log logr.Logger, i *networkingv1alpha1.SwitchInterface) (ctrl.Result, error) {
	log.Info("Deleting SwitchInterface")

	// TODO: do cleanup

	if _, err := clientutils.PatchEnsureNoFinalizer(ctx, r.Client, i, networkingv1alpha1.SwitchFinalizer); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Deleted SwitchInterface")
	return ctrl.Result{}, nil
}

func (r *SwitchInterfaceReconciler) reconcile(ctx context.Context, log logr.Logger, i *networkingv1alpha1.SwitchInterface) (ctrl.Result, error) {
	log.Info("Reconciling SwitchInterface")

	if modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, i, networkingv1alpha1.SwitchFinalizer); err != nil || modified {
		return ctrl.Result{}, err
	}

	// No device backend is configured to observe or apply interface state.
	// Keep the resource pending rather than claiming its desired state is applied.
	if i.Status.State == networkingv1alpha1.SwitchInterfaceStatePending {
		return ctrl.Result{}, nil
	}

	base := client.MergeFrom(i.DeepCopy())
	i.Status.State = networkingv1alpha1.SwitchInterfaceStatePending
	if err := r.Status().Patch(ctx, i, base); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciled SwitchInterface")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwitchInterfaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.SwitchInterface{}).
		Named("switchinterface").
		Complete(r)
}

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

// SwitchReconciler reconciles a Switch object
type SwitchReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=switches,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=switches/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=switches/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SwitchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	s := &networkingv1alpha1.Switch{}
	if err := r.Get(ctx, req.NamespacedName, s); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconileExists(ctx, log, s)
}

func (r *SwitchReconciler) reconileExists(ctx context.Context, log logr.Logger, s *networkingv1alpha1.Switch) (ctrl.Result, error) {
	if !s.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, s)
	}
	return r.reconcile(ctx, log, s)
}

func (r *SwitchReconciler) delete(ctx context.Context, log logr.Logger, s *networkingv1alpha1.Switch) (ctrl.Result, error) {
	log.Info("Deleting Switch")

	// TODO: do cleanup

	if _, err := clientutils.PatchEnsureNoFinalizer(ctx, r.Client, s, networkingv1alpha1.SwitchFinalizer); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Deleted Switch")
	return ctrl.Result{}, nil
}

func (r *SwitchReconciler) reconcile(ctx context.Context, log logr.Logger, s *networkingv1alpha1.Switch) (ctrl.Result, error) {
	log.Info("Reconciling Switch")

	if modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, s, networkingv1alpha1.SwitchFinalizer); err != nil || modified {
		return ctrl.Result{}, err
	}

	// Readiness reflects declarative acceptance for ZTP/bootstrap workflows.
	// Device discovery and status observation require a supported backend.
	if s.Status.State == networkingv1alpha1.SwitchStateReady {
		return ctrl.Result{}, nil
	}

	base := client.MergeFrom(s.DeepCopy())
	s.Status.State = networkingv1alpha1.SwitchStateReady
	if err := r.Status().Patch(ctx, s, base); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciled Switch")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwitchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.Switch{}).
		Named("switch").
		Complete(r)
}

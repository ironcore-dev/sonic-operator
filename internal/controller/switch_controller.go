// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/controller-utils/clientutils"

	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"

	switchUtil "github.com/ironcore-dev/sonic-operator/internal/switch_util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
)

var (
	fieldOwner = client.FieldOwner("switch-controller")
)

// SwitchReconciler reconciles a Switch object
type SwitchReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=switches,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=switches/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=switches/finalizers,verbs=update
// +kubebuilder:rbac:groups=sonic.networking.metal.ironcore.dev,resources=interfaces,verbs=get;list;watch;create;update;patch;delete

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

	original := s.DeepCopy()
	defer func() {
		if err := r.Status().Patch(ctx, s, client.MergeFrom(original)); err != nil {
			log.Error(err, "Failed to update Switch status")
		}
	}()

	if s.Status.State == "" {
		s.Status.State = networkingv1alpha1.SwitchStatePending
		return ctrl.Result{}, nil
	}

	switchAgentClient, err := switchUtil.NewAgentClientForSwitch(ctx, s)
	if err != nil {
		return ctrl.Result{}, err
	}

	switchDevice, err := switchAgentClient.GetDeviceInfo(ctx)
	if err != nil {
		s.Status.State = networkingv1alpha1.SwitchStateFailed
		return ctrl.Result{}, err
	}

	s.Status.MACAddress = switchDevice.LocalMacAddress
	s.Status.FirmwareVersion = switchDevice.SonicOSVersion
	s.Status.SKU = switchDevice.Hwsku

	interfaceList, err := switchAgentClient.ListInterfaces(ctx)
	if err != nil {
		s.Status.State = networkingv1alpha1.SwitchStateFailed
		return ctrl.Result{}, err
	}

	for _, iface := range interfaceList.Items {
		if err := r.EnsureInterface(ctx, log, s, iface); err != nil {
			return ctrl.Result{}, err
		}
	}

	portList, err := switchAgentClient.ListPorts(ctx)
	if err != nil {
		s.Status.State = networkingv1alpha1.SwitchStateFailed
		return ctrl.Result{}, err
	}

	if len(portList.Items) > 0 {
		s.Status.Ports = make([]networkingv1alpha1.PortStatus, 0, len(portList.Items))
	}

	for _, p := range portList.Items {
		s.Status.Ports = append(s.Status.Ports, networkingv1alpha1.PortStatus{Name: p.Name})
	}

	s.Status.State = networkingv1alpha1.SwitchStateReady

	// TODO: ensure s.spec is applied

	log.Info("Reconciled Switch")
	return ctrl.Result{}, nil
}

func (r *SwitchReconciler) EnsureInterface(ctx context.Context, log logr.Logger, s *networkingv1alpha1.Switch, iface agent.Interface) error {
	log.Info("Ensuring Interface")

	adminState, err := agent.AgentDeviceStatusToAPIAdminState(iface.AdminStatus)
	if err != nil {
		return err
	}
	i := &networkingv1alpha1.SwitchInterface{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkingv1alpha1.GroupVersion.String(),
			Kind:       "SwitchInterface",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.ToLower(iface.Name),
		},
		Spec: networkingv1alpha1.SwitchInterfaceSpec{
			Handle: iface.Name,
			SwitchRef: &corev1.LocalObjectReference{
				Name: s.Name,
			},
			AdminState: adminState,
		},
	}

	if err := controllerutil.SetOwnerReference(s, i, r.Scheme); err != nil {
		return err
	}

	if err := r.Patch(ctx, i, client.Apply, client.ForceOwnership, fieldOwner); err != nil {
		return err
	}

	log.Info("Ensured Interface")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwitchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.Switch{}).
		Owns(&networkingv1alpha1.SwitchInterface{}).
		Named("switch").
		Complete(r)
}

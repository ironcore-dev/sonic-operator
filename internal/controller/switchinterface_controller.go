// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
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
	agenterrors "github.com/ironcore-dev/sonic-operator/internal/agent/errors"
	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"
	switchUtil "github.com/ironcore-dev/sonic-operator/internal/switch_util"
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

	original := i.DeepCopy()
	defer func() {
		if err := r.Status().Patch(ctx, i, client.MergeFrom(original)); err != nil {
			log.Error(err, "Failed to update Switch status")
		}
	}()

	if i.Status.State == "" {
		i.Status.State = networkingv1alpha1.SwitchInterfaceStatePending
		return ctrl.Result{}, nil
	}

	if i.Spec.SwitchRef == nil {
		i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
		return ctrl.Result{}, nil
	}

	switchAgentClient, err := switchUtil.NewAgentClientFromSwitchRef(ctx, r.Client, i.Spec.SwitchRef, i.Namespace)
	if err != nil {
		i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
		return ctrl.Result{}, err
	}

	iface, err := switchAgentClient.GetInterface(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name: i.Spec.Handle,
	})
	if err != nil {
		i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
		return ctrl.Result{}, err
	}

	if iface != nil {
		if iface.OperationStatus == agent.StatusUp {
			i.Status.OperationalState = networkingv1alpha1.OperationStateUp
		} else {
			i.Status.OperationalState = networkingv1alpha1.OperationStateDown
		}

		adminState, err := agent.AgentDeviceStatusToAPIAdminState(iface.AdminStatus)
		if err != nil {
			i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
			return ctrl.Result{}, err
		}
		i.Status.AdminState = adminState
	} else {
		i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
		return ctrl.Result{}, nil
	}

	// ensure i.spec.AdminState is applied
	desired_state, err := agent.APIAdminStateToAgentDeviceStatus(i.Spec.AdminState)
	if err != nil {
		i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
		return ctrl.Result{}, err
	}

	var switchInterface *agent.Interface
	if switchInterface, err = switchAgentClient.SetInterfaceAdminStatus(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name:        i.Spec.Handle,
		AdminStatus: desired_state,
	}); err != nil {
		i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
		return ctrl.Result{}, err
	}

	if switchInterface != nil {
		adminState, err := agent.AgentDeviceStatusToAPIAdminState(switchInterface.AdminStatus)
		if err != nil {
			i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
			return ctrl.Result{}, err
		}
		i.Status.AdminState = adminState

		operationState, err := agent.AgentDeviceStatusToAPIOperationState(switchInterface.OperationStatus)
		if err != nil {
			i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
			return ctrl.Result{}, err
		}
		i.Status.OperationalState = operationState

		if switchInterface.OperationStatus == agent.StatusUp {
			i.Status.OperationalState = networkingv1alpha1.OperationStateUp
		} else {
			i.Status.OperationalState = networkingv1alpha1.OperationStateDown
		}
	}
	i.Status.State = networkingv1alpha1.SwitchInterfaceStateReady

	neighbor, err := switchAgentClient.GetInterfaceNeighbor(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name: i.Spec.Handle,
	})
	if err != nil {
		if neighbor == nil || neighbor.Status.Code != agenterrors.NOT_FOUND {
			i.Status.State = networkingv1alpha1.SwitchInterfaceStateFailed
			return ctrl.Result{}, err
		}
		i.Status.Neighbor = networkingv1alpha1.Neighbor{}
	} else {
		i.Status.Neighbor = networkingv1alpha1.Neighbor{
			MacAddress:      neighbor.MacAddress,
			SystemName:      neighbor.SystemName,
			InterfaceHandle: neighbor.Handle,
		}
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

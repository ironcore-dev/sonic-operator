// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package switchutil

import (
	"context"

	networkingv1alpha1 "github.com/ironcore-dev/switch-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentCli "github.com/ironcore-dev/switch-operator/internal/agent/agent_client/client"
)

func NewAgentClientForSwitch(ctx context.Context, s *networkingv1alpha1.Switch) (agentCli.SwitchAgentClient, error) {
	// TODO: construct client from s.spec.Management

	if s.Spec.Management.Host == "" && s.Spec.Management.Port == "" {
		agentcli, err := agentCli.NewDefaultSwitchAgentClient("", 0)
		return agentcli, err
	}

	address := s.Spec.Management.Host + ":" + s.Spec.Management.Port

	agentcli, err := agentCli.NewDefaultSwitchAgentClient(address, 0)
	if err != nil {
		return nil, err
	}

	return agentcli, nil
}

func NewAgentClientFromSwitchRef(ctx context.Context, cli client.Reader, ref *v1.LocalObjectReference, nameSpace string) (agentCli.SwitchAgentClient, error) {
	if ref == nil {
		return nil, nil
	}

	ownerSwitch := &networkingv1alpha1.Switch{}
	err := cli.Get(ctx, types.NamespacedName{
		Name:      ref.Name,
		Namespace: nameSpace,
	}, ownerSwitch)

	if err != nil {
		return nil, err
	}

	agentcli, err := NewAgentClientForSwitch(ctx, ownerSwitch)
	if err != nil {
		return nil, err
	}

	return agentcli, nil
}

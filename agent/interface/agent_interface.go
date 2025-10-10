// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"

	agent "github.com/ironcore-dev/switch-operator/agent/types"
)

type SwitchAgent interface {
	GetDeviceInfo(ctx context.Context) (*agent.SwitchDevice, *agent.Status)
	ListInterfaces(ctx context.Context) (*agent.InterfaceList, *agent.Status)

	SetInterfaceAdminStatus(ctx context.Context, iface *agent.Interface) (*agent.Interface, *agent.Status)
	GetInterface(ctx context.Context, iface *agent.Interface) (*agent.Interface, *agent.Status)
	GetInterfaceNeighbor(ctx context.Context, iface *agent.Interface) (*agent.InterfaceNeighbor, *agent.Status)

	ListPorts(ctx context.Context) (*agent.PortList, *agent.Status)
}

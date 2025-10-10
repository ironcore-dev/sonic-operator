// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"

	networkingv1alpha1 "github.com/ironcore-dev/switch-operator/api/v1alpha1"
)

type Agent struct {
}

type SwitchInfo struct {
	FirmwareVersion string `json:"firmwareVersion"`
	SKU             string `json:"sku"`
	MACAddress      string `json:"macAddress"`
}

func (a *Agent) GetSwitchInfo(ctx context.Context) (*SwitchInfo, error) {
	return &SwitchInfo{
		FirmwareVersion: "1.0.0",
		SKU:             "SKU12345",
		MACAddress:      "00:11:22:33:44:55",
	}, nil
}

type Port struct {
	Name string `json:"name"`
}

func (a *Agent) ListPorts(ctx context.Context) ([]Port, error) {
	var ports []Port
	ports = append(ports, Port{Name: "port0"})
	ports = append(ports, Port{Name: "port1"})
	return ports, nil
}

type Interfaces struct {
	Name       string `json:"name"`
	Handle     string `json:"handle"`
	AdminState string `json:"adminState"`
}

func (a *Agent) GetInterfaces(ctx context.Context) ([]Interfaces, error) {
	var interfaces []Interfaces
	interfaces = append(interfaces, Interfaces{Name: "eth0"})
	return interfaces, nil
}

func (*Agent) GetInterface(ctx context.Context, name string) (Interfaces, error) {
	return Interfaces{
		Name:       "foo",
		Handle:     "bar",
		AdminState: "Up",
	}, nil
}

func NewAgentClientForSwitch(ctx context.Context, s *networkingv1alpha1.Switch) (Agent, error) {
	// TODO: construct client from s.spec.Management

	return Agent{}, nil
}

func NewAgentClientForInterface(ctx context.Context, i *networkingv1alpha1.SwitchInterface) (Agent, error) {
	return Agent{}, nil
}

// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/ironcore-dev/sonic-operator/internal/agent/proto"
	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"
)

type SwitchAgentClient interface {
	GetDeviceInfo(ctx context.Context) (*agent.SwitchDevice, error)
	ListInterfaces(ctx context.Context) (*agent.InterfaceList, error)
	GetInterfaceByAbstractName(ctx context.Context, iface *agent.Interface) (*agent.Interface, error)

	GetInterfaceNeighbor(ctx context.Context, iface *agent.Interface) (*agent.InterfaceNeighbor, error)

	SetInterfaceAdminStatus(ctx context.Context, iface *agent.Interface) (*agent.Interface, error)
	SetInterfaceAliasName(ctx context.Context, iface *agent.Interface) (*agent.Interface, error)

	ListPorts(ctx context.Context) (*agent.PortList, error)

	SaveConfig(ctx context.Context) error
}

type defaultSwitchAgentClient struct {
	Address        string
	ConnectTimeout time.Duration

	opts   []grpc.DialOption // Options for the gRPC connection
	client pb.SwitchAgentServiceClient
}

func NewDefaultSwitchAgentClient(address string, connectTimeout time.Duration) (SwitchAgentClient, error) {
	if address == "" {
		address = "localhost:50051"
	}

	if connectTimeout == 0 {
		connectTimeout = 4 * time.Second
	}

	c := defaultSwitchAgentClient{
		Address:        address,
		ConnectTimeout: connectTimeout,
	}

	// Remove the println from here - flags haven't been parsed yet!
	c.opts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	return &c, nil
}

func (c *defaultSwitchAgentClient) dial() (func() error, error) {
	println("connect to ", c.Address)

	conn, err := grpc.NewClient(c.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// conn, err := grpc.DialContext(dialCtx, c.Address,
	// 	grpc.WithTransportCredentials(insecure.NewCredentials()),
	// 	grpc.WithBlock(), // Wait for connection to be ready
	// )
	if err != nil {
		return nil, fmt.Errorf("failed to connect to switch proxy: %w", err)
	}

	c.client = pb.NewSwitchAgentServiceClient(conn)

	// Return a cleanup function that ensures proper connection termination
	return func() error {
		return conn.Close()
	}, nil
}

func (c *defaultSwitchAgentClient) GetDeviceInfo(ctx context.Context) (*agent.SwitchDevice, error) {
	cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := c.client.GetDeviceInfo(ctx, &pb.GetDeviceInfoRequest{})
	if err != nil {
		return nil, err
	}

	device := &agent.SwitchDevice{
		TypeMeta: agent.TypeMeta{
			Kind: agent.DeviceKind,
		},
		LocalMacAddress: resp.GetLocalMacAddress(),
		Hwsku:           resp.GetHwsku(),
		SonicOSVersion:  resp.GetSonicOsVersion(),
		AsicType:        resp.GetAsicType(),
		Readiness:       resp.GetReadiness(),
		Status:          agent.ProtoStatusToStatus(resp.GetStatus()),
	}

	return device, nil
}

func (c *defaultSwitchAgentClient) ListInterfaces(ctx context.Context) (*agent.InterfaceList, error) {
	cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := c.client.ListInterfaces(ctx, &pb.ListInterfacesRequest{})
	if err != nil {
		return nil, err
	}

	interfaces := make([]agent.Interface, len(resp.GetInterfaces()))
	for i, iface := range resp.GetInterfaces() {
		interfaces[i] = agent.Interface{
			TypeMeta: agent.TypeMeta{
				Kind: agent.InterfaceKind,
			},
			Name:            iface.GetName(),
			NativeName:      iface.GetNativeName(),
			AliasName:       iface.GetAliasName(),
			MacAddress:      iface.GetMacAddress(),
			OperationStatus: agent.DeviceStatus(iface.GetOperationalStatus()),
			AdminStatus:     agent.DeviceStatus(iface.GetAdminStatus()),
		}
	}

	interfaceList := &agent.InterfaceList{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceListKind,
		},
		Items:  interfaces,
		Status: agent.ProtoStatusToStatus(resp.GetStatus()),
	}

	return interfaceList, nil
}

func (c *defaultSwitchAgentClient) SetInterfaceAdminStatus(ctx context.Context, iface *agent.Interface) (*agent.Interface, error) {
	cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := c.client.SetInterfaceAdminStatus(ctx, &pb.SetInterfaceAdminStatusRequest{
		InterfaceName: iface.GetName(),
		AdminStatus:   string(iface.AdminStatus),
	})
	if err != nil {
		fmt.Println("Error occurred while setting interface admin status:", err)
		return nil, err
	}

	if resp.GetStatus().Code != 0 {
		fmt.Println("Error occurred while setting interface admin status:", resp.GetStatus().GetMessage())
		return &agent.Interface{
			Status: agent.ProtoStatusToStatus(resp.GetStatus()),
		}, fmt.Errorf("failed to set interface admin status: %s", resp.GetStatus().GetMessage())
	}
	iface.Name = resp.GetInterface().GetName()
	iface.AliasName = resp.GetInterface().GetAliasName()
	iface.NativeName = resp.GetInterface().GetNativeName()
	iface.MacAddress = resp.GetInterface().GetMacAddress()
	iface.AdminStatus = agent.DeviceStatus(resp.GetInterface().GetAdminStatus())
	iface.OperationStatus = agent.DeviceStatus(resp.GetInterface().GetOperationalStatus())
	iface.Status = agent.ProtoStatusToStatus(resp.GetStatus())

	return iface, nil
}

func (c *defaultSwitchAgentClient) GetInterfaceByAbstractName(ctx context.Context, iface *agent.Interface) (*agent.Interface, error) {
	cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	nativeName, err := agent.AbstractNameToNativeName(iface.GetName())
	if err != nil {
		return nil, err
	}

	resp, err := c.client.GetInterface(ctx, &pb.GetInterfaceRequest{
		InterfaceName: nativeName,
	})
	if err != nil {
		return nil, err
	}

	if resp.GetStatus().Code != 0 {
		return &agent.Interface{
			Status: agent.ProtoStatusToStatus(resp.GetStatus()),
		}, fmt.Errorf("failed to get interface: %s", resp.GetStatus().GetMessage())
	}

	return &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name:            resp.GetInterface().Name,
		AliasName:       resp.GetInterface().AliasName,
		NativeName:      resp.GetInterface().NativeName,
		MacAddress:      resp.GetInterface().GetMacAddress(),
		OperationStatus: agent.DeviceStatus(resp.GetInterface().GetOperationalStatus()),
		AdminStatus:     agent.DeviceStatus(resp.GetInterface().GetAdminStatus()),
		Status:          agent.ProtoStatusToStatus(resp.GetStatus()),
	}, nil
}

func (c *defaultSwitchAgentClient) GetInterfaceNeighbor(ctx context.Context, iface *agent.Interface) (*agent.InterfaceNeighbor, error) {
	cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := c.client.GetInterfaceNeighbor(ctx, &pb.GetInterfaceNeighborRequest{
		InterfaceName: iface.GetName(),
	})
	if err != nil {
		return nil, err
	}

	if resp.GetStatus().Code != 0 {
		return &agent.InterfaceNeighbor{
			Status: agent.ProtoStatusToStatus(resp.GetStatus()),
		}, fmt.Errorf("failed to get interface neighbor: %s", resp.GetStatus().GetMessage())
	}

	return &agent.InterfaceNeighbor{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceNeighborKind,
		},
		Name:       resp.GetInterface(),
		MacAddress: resp.GetNeighbor().GetMacAddress(),
		SystemName: resp.GetNeighbor().GetSystemName(),
		Handle:     resp.GetNeighbor().GetNeighborInterfaceName(),
		Status:     agent.ProtoStatusToStatus(resp.GetStatus()),
	}, nil
}

func (c *defaultSwitchAgentClient) ListPorts(ctx context.Context) (*agent.PortList, error) {
	cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := c.client.ListPorts(ctx, &pb.ListPortsRequest{})
	if err != nil {
		return nil, err
	}

	ports := make([]agent.Port, len(resp.GetPorts()))
	for i, port := range resp.GetPorts() {
		ports[i] = agent.Port{
			TypeMeta: agent.TypeMeta{
				Kind: agent.PortKind,
			},
			Name:  port.GetName(),
			Alias: port.GetAlias(),
		}
	}

	portList := &agent.PortList{
		TypeMeta: agent.TypeMeta{
			Kind: agent.PortListKind,
		},
		Items:  ports,
		Status: agent.ProtoStatusToStatus(resp.GetStatus()),
	}

	return portList, nil
}

func (c *defaultSwitchAgentClient) SetInterfaceAliasName(ctx context.Context, iface *agent.Interface) (*agent.Interface, error) {
	cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := c.client.SetInterfaceAliasName(ctx, &pb.SetInterfaceAliasNameRequest{
		InterfaceName: iface.GetName(),
		AliasName:     iface.AliasName,
	})
	if err != nil {
		fmt.Println("Error occurred while setting interface alias name:", err)
		return nil, err
	}

	if resp.GetStatus().Code != 0 {
		fmt.Println("Error occurred while setting interface alias name:", resp.GetStatus().GetMessage())
		return &agent.Interface{
			Status: agent.ProtoStatusToStatus(resp.GetStatus()),
		}, fmt.Errorf("failed to set interface alias name: %s", resp.GetStatus().GetMessage())
	}

	iface.AdminStatus = agent.DeviceStatus(resp.GetInterface().GetAdminStatus())
	iface.OperationStatus = agent.DeviceStatus(resp.GetInterface().GetOperationalStatus())
	iface.Status = agent.ProtoStatusToStatus(resp.GetStatus())

	return iface, nil
}

func (c *defaultSwitchAgentClient) SaveConfig(ctx context.Context) error {
	cleanup, err := c.dial()
	if err != nil {
		return err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := c.client.SaveConfig(ctx, &pb.SaveConfigRequest{})
	if err != nil {
		return err
	}

	if resp.GetStatus().Code != 0 {
		return fmt.Errorf("failed to save config: %s", resp.GetStatus().GetMessage())
	}

	return nil
}

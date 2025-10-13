// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/ironcore-dev/switch-operator/agent/proto"
	agent "github.com/ironcore-dev/switch-operator/agent/types"
)

type SwitchAgentClient interface {
	AddSwitchAgentClientFlags(fs *pflag.FlagSet)

	GetDeviceInfo(ctx context.Context) (*agent.SwitchDevice, error)
	ListInterfaces(ctx context.Context) (*agent.InterfaceList, error)
	GetInterface(ctx context.Context, iface *agent.Interface) (*agent.Interface, error)

	SetInterfaceAdminStatus(ctx context.Context, iface *agent.Interface) (*agent.Interface, error)

	ListPorts(ctx context.Context) (*agent.PortList, error)
}

type defaultSwitchAgentClient struct {
	Address        string
	ConnectTimeout time.Duration

	opts   []grpc.DialOption // Options for the gRPC connection
	client pb.SwitchAgentServiceClient
}

func (c *defaultSwitchAgentClient) AddSwitchAgentClientFlags(fs *pflag.FlagSet) {
	grpcPort := os.Getenv("SWITCH_PROXY_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	fs.StringVar(&c.Address, "address", "localhost:"+grpcPort, "switch proxy address (overrides SWITCH_PROXY_GRPC_PORT).")
	fs.DurationVar(&c.ConnectTimeout, "connect-timeout", 4*time.Second, "Timeout to connect to the switch proxy.")
}

func NewDefaultSwitchAgentClient() (SwitchAgentClient, error) {
	var c defaultSwitchAgentClient

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
			MacAddress:      iface.GetMacAddress(),
			OperationStatus: iface.GetOperationalStatus(),
			AdminStatus:     iface.GetAdminStatus(),
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
		AdminStatus:   iface.AdminStatus,
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

	return iface, nil
}

func (c *defaultSwitchAgentClient) GetInterface(ctx context.Context, iface *agent.Interface) (*agent.Interface, error) {
	cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := c.client.GetInterface(ctx, &pb.GetInterfaceRequest{
		InterfaceName: iface.GetName(),
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
		Name:            resp.GetInterface().GetName(),
		MacAddress:      resp.GetInterface().GetMacAddress(),
		OperationStatus: resp.GetInterface().GetOperationalStatus(),
		AdminStatus:     resp.GetInterface().GetAdminStatus(),
		Status:          agent.ProtoStatusToStatus(resp.GetStatus()),
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

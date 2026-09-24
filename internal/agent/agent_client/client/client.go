// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"
	pb "github.com/ironcore-dev/sonic-operator/pkg/agent/proto"
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
	Reboot(ctx context.Context) error
	OnieBootModeInstall(ctx context.Context) error
	RestartSystemdService(ctx context.Context, serviceName string) error
	RebootCause(ctx context.Context) (string, error)
	FactoryReset(ctx context.Context) error
	GetReadiness(ctx context.Context) (bool, error)
	Reprovision(ctx context.Context) error

	ApplySwitch(ctx context.Context, device string, cfg *pb.SwitchConfig) error
	// DeleteSwitch triggers async reprovision. Returns the state string:
	// "" = done (or not yet started), "Reprovisioning" = in progress.
	DeleteSwitch(ctx context.Context, device string) (state string, err error)
	// EnsureReprovision calls DeleteSwitch in a loop until state is "" (done).
	EnsureReprovision(ctx context.Context, device string) error
}

type defaultSwitchAgentClient struct {
	Address        string
	ConnectTimeout time.Duration
	plainMode      bool
}

func NewDefaultSwitchAgentClient(address string, connectTimeout time.Duration, plainMode bool) (SwitchAgentClient, error) {
	if address == "" {
		address = "localhost:50051"
	}

	if connectTimeout == 0 {
		connectTimeout = 4 * time.Second
	}

	c := defaultSwitchAgentClient{
		Address:        address,
		ConnectTimeout: connectTimeout,
		plainMode:      plainMode,
	}

	return &c, nil
}

func (c *defaultSwitchAgentClient) dial() (pb.SwitchAgentServiceClient, func() error, error) {
	log.Printf("connecting to %s", c.Address)

	var creds grpc.DialOption
	if c.plainMode {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		creds = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})) //nolint:gosec
	}

	conn, err := grpc.NewClient(c.Address, creds)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to switch proxy: %w", err)
	}

	grpcClient := pb.NewSwitchAgentServiceClient(conn)

	// Return a cleanup function that ensures proper connection termination
	return grpcClient, func() error {
		return conn.Close()
	}, nil
}

func (c *defaultSwitchAgentClient) dialConn() (*grpc.ClientConn, error) {
	var creds grpc.DialOption
	if c.plainMode {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		creds = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})) //nolint:gosec
	}
	conn, err := grpc.NewClient(c.Address, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to switch proxy: %w", err)
	}
	return conn, nil
}

func (c *defaultSwitchAgentClient) dialFabricSonic() (pb.FabricSonicServiceClient, func() error, error) {
	log.Printf("connecting to %s", c.Address)
	conn, err := c.dialConn()
	if err != nil {
		return nil, nil, err
	}
	return pb.NewFabricSonicServiceClient(conn), conn.Close, nil
}

func (c *defaultSwitchAgentClient) GetDeviceInfo(ctx context.Context) (*agent.SwitchDevice, error) {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.GetDeviceInfo(ctx, &pb.GetDeviceInfoRequest{})
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
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.ListInterfaces(ctx, &pb.ListInterfacesRequest{})
	if err != nil {
		return nil, err
	}

	interfaces := make([]agent.Interface, len(resp.GetInterfaces()))
	for i, iface := range resp.GetInterfaces() {
		interfaces[i] = agent.Interface{
			TypeMeta: agent.TypeMeta{
				Kind: agent.InterfaceKind,
			},
			NativeName:      iface.GetName(),
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
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.SetInterfaceAdminStatus(ctx, &pb.SetInterfaceAdminStatusRequest{
		InterfaceName: iface.GetName(),
		AdminStatus:   string(iface.AdminStatus),
	})
	if err != nil {
		return nil, err
	}

	if resp.GetStatus().Code != 0 {
		return &agent.Interface{
			Status: agent.ProtoStatusToStatus(resp.GetStatus()),
		}, fmt.Errorf("failed to set interface admin status: %s", resp.GetStatus().GetMessage())
	}
	iface.NativeName = resp.GetInterface().GetName()
	iface.AliasName = resp.GetInterface().GetAliasName()
	iface.MacAddress = resp.GetInterface().GetMacAddress()
	iface.AdminStatus = agent.DeviceStatus(resp.GetInterface().GetAdminStatus())
	iface.OperationStatus = agent.DeviceStatus(resp.GetInterface().GetOperationalStatus())
	iface.Status = agent.ProtoStatusToStatus(resp.GetStatus())

	return iface, nil
}

func (c *defaultSwitchAgentClient) GetInterfaceByAbstractName(ctx context.Context, iface *agent.Interface) (*agent.Interface, error) {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.GetInterface(ctx, &pb.GetInterfaceRequest{
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
		NativeName:      resp.GetInterface().GetName(),
		AliasName:       resp.GetInterface().GetAliasName(),
		MacAddress:      resp.GetInterface().GetMacAddress(),
		OperationStatus: agent.DeviceStatus(resp.GetInterface().GetOperationalStatus()),
		AdminStatus:     agent.DeviceStatus(resp.GetInterface().GetAdminStatus()),
		Status:          agent.ProtoStatusToStatus(resp.GetStatus()),
	}, nil
}

func (c *defaultSwitchAgentClient) GetInterfaceNeighbor(ctx context.Context, iface *agent.Interface) (*agent.InterfaceNeighbor, error) {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.GetInterfaceNeighbor(ctx, &pb.GetInterfaceNeighborRequest{
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
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.ListPorts(ctx, &pb.ListPortsRequest{})
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
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.SetInterfaceAliasName(ctx, &pb.SetInterfaceAliasNameRequest{
		InterfaceName: iface.GetName(),
		AliasName:     iface.AliasName,
	})
	if err != nil {
		return nil, err
	}

	if resp.GetStatus().Code != 0 {
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
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.SaveConfig(ctx, &pb.SaveConfigRequest{})
	if err != nil {
		return err
	}

	if resp.GetStatus().Code != 0 {
		return fmt.Errorf("failed to save config: %s", resp.GetStatus().GetMessage())
	}

	return nil
}

func (c *defaultSwitchAgentClient) Reboot(ctx context.Context) error {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.Reboot(ctx, &pb.RebootRequest{})
	if err != nil {
		return err
	}

	if resp.GetStatus().Code != 0 {
		return fmt.Errorf("failed to reboot switch: %s", resp.GetStatus().GetMessage())
	}

	return nil
}

func (c *defaultSwitchAgentClient) OnieBootModeInstall(ctx context.Context) error {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.OnieBootModeInstall(ctx, &pb.OnieBootModeInstallRequest{})
	if err != nil {
		return err
	}

	if resp.GetStatus().Code != 0 {
		return fmt.Errorf("failed to set ONIE boot mode: %s", resp.GetStatus().GetMessage())
	}

	return nil
}

func (c *defaultSwitchAgentClient) RestartSystemdService(ctx context.Context, serviceName string) error {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.RestartSystemdService(ctx, &pb.RestartSystemdServiceRequest{
		ServiceName: serviceName,
	})
	if err != nil {
		return err
	}

	if resp.GetStatus().Code != 0 {
		return fmt.Errorf("failed to restart systemd service %s: %s", serviceName, resp.GetStatus().GetMessage())
	}

	return nil
}

func (c *defaultSwitchAgentClient) RebootCause(ctx context.Context) (string, error) {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.RebootCause(ctx, &pb.RebootCauseRequest{})
	if err != nil {
		return "", err
	}

	if resp.GetStatus().Code != 0 {
		return "", fmt.Errorf("failed to get reboot cause: %s", resp.GetStatus().GetMessage())
	}

	return resp.GetCause(), nil
}

func (c *defaultSwitchAgentClient) FactoryReset(ctx context.Context) error {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.FactoryReset(ctx, &pb.FactoryResetRequest{})
	if err != nil {
		return err
	}

	if resp.GetStatus().Code != 0 {
		return fmt.Errorf("failed to factory reset: %s", resp.GetStatus().GetMessage())
	}

	return nil
}

func (c *defaultSwitchAgentClient) GetReadiness(ctx context.Context) (bool, error) {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return false, err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.GetReadiness(ctx, &pb.GetReadinessRequest{})
	if err != nil {
		return false, err
	}

	if resp.GetStatus().Code != 0 {
		return false, fmt.Errorf("failed to get readiness: %s", resp.GetStatus().GetMessage())
	}

	return resp.GetReady(), nil
}

func (c *defaultSwitchAgentClient) Reprovision(ctx context.Context) error {
	grpcClient, cleanup, err := c.dial()
	if err != nil {
		return err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := grpcClient.Reprovision(ctx, &pb.ReprovisionRequest{})
	if err != nil {
		return err
	}

	if status := resp.GetStatus(); status != nil && status.Code != 0 {
		return fmt.Errorf("failed to reprovision: %s", status.GetMessage())
	}

	return nil
}

func (c *defaultSwitchAgentClient) ApplySwitch(ctx context.Context, device string, cfg *pb.SwitchConfig) error {
	fabricClient, cleanup, err := c.dialFabricSonic()
	if err != nil {
		return err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := fabricClient.ApplySwitch(ctx, &pb.ApplySwitchRequest{
		Device: device,
		Config: cfg,
	})
	if err != nil {
		return fmt.Errorf("ApplySwitch RPC failed: %w", err)
	}
	_ = resp
	return nil
}

func (c *defaultSwitchAgentClient) DeleteSwitch(ctx context.Context, device string) (string, error) {
	fabricClient, cleanup, err := c.dialFabricSonic()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = cleanup()
	}()

	resp, err := fabricClient.DeleteSwitch(ctx, &pb.DeleteSwitchRequest{Device: device})
	if err != nil {
		return "", fmt.Errorf("DeleteSwitch RPC failed: %w", err)
	}
	return resp.GetState(), nil
}

func (c *defaultSwitchAgentClient) EnsureReprovision(ctx context.Context, device string) error {
	for {
		state, err := c.DeleteSwitch(ctx, device)
		if err != nil {
			return err
		}
		if state == "" {
			return nil
		}
		t := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
}

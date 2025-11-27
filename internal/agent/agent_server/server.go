// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package agent_server

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/ironcore-dev/switch-operator/internal/agent/proto"
	agent "github.com/ironcore-dev/switch-operator/internal/agent/types"

	switchAgent "github.com/ironcore-dev/switch-operator/internal/agent/interface"
	"github.com/ironcore-dev/switch-operator/internal/agent/sonic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port      = flag.Int("port", 50051, "The server port")
	redisAddr = flag.String("redis-addr", "127.0.0.1:6379", "The Redis address")
)

type proxyServer struct {
	pb.UnimplementedSwitchAgentServiceServer

	SwitchAgent switchAgent.SwitchAgent
}

func (s *proxyServer) GetDeviceInfo(ctx context.Context, request *pb.GetDeviceInfoRequest) (*pb.GetDeviceInfoResponse, error) {
	// Fetch device info from the SwitchAgent
	device, status := s.SwitchAgent.GetDeviceInfo(ctx)
	if status != nil {
		return &pb.GetDeviceInfoResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: status.Message,
			},
		}, nil
	}

	return &pb.GetDeviceInfoResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
		LocalMacAddress: device.LocalMacAddress,
		Hwsku:           device.Hwsku,
		SonicOsVersion:  device.SonicOSVersion,
		AsicType:        device.AsicType,
		Readiness:       device.Readiness,
	}, nil
}

func (s *proxyServer) ListInterfaces(ctx context.Context, request *pb.ListInterfacesRequest) (*pb.ListInterfacesResponse, error) {
	interfaceList, status := s.SwitchAgent.ListInterfaces(ctx)
	if status != nil {
		return &pb.ListInterfacesResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: fmt.Sprintf("failed to list interfaces: %v", status.Message),
			},
		}, nil
	}

	var interfaces = make([]*pb.Interface, 0, len(interfaceList.Items))
	for _, iface := range interfaceList.Items {
		interfaces = append(interfaces, &pb.Interface{
			Name:              iface.Name,
			MacAddress:        iface.MacAddress,
			OperationalStatus: string(iface.OperationStatus),
			AdminStatus:       string(iface.AdminStatus),
		})
	}

	return &pb.ListInterfacesResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
		Interfaces: interfaces,
	}, nil
}

func (s *proxyServer) SetInterfaceAdminStatus(ctx context.Context, request *pb.SetInterfaceAdminStatusRequest) (*pb.SetInterfaceAdminStatusResponse, error) {
	iface, status := s.SwitchAgent.SetInterfaceAdminStatus(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name:        request.GetInterfaceName(),
		AdminStatus: agent.DeviceStatus(request.GetAdminStatus()),
	})

	if status != nil {
		return &pb.SetInterfaceAdminStatusResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: status.Message,
			},
		}, nil
	}

	return &pb.SetInterfaceAdminStatusResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
		Interface: &pb.Interface{
			Name:              iface.Name,
			MacAddress:        "",
			OperationalStatus: string(iface.OperationStatus),
			AdminStatus:       string(iface.AdminStatus),
		},
	}, nil
}

func (s *proxyServer) ListPorts(ctx context.Context, request *pb.ListPortsRequest) (*pb.ListPortsResponse, error) {
	portList, status := s.SwitchAgent.ListPorts(ctx)
	if status != nil {
		return &pb.ListPortsResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: fmt.Sprintf("failed to list ports: %v", status.Message),
			},
		}, nil
	}

	var ports = make([]*pb.Port, 0, len(portList.Items))
	for _, port := range portList.Items {
		ports = append(ports, &pb.Port{
			Name:  port.Name,
			Alias: port.Alias,
		})
	}

	return &pb.ListPortsResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
		Ports: ports,
	}, nil
}

func (s *proxyServer) GetInterface(ctx context.Context, request *pb.GetInterfaceRequest) (*pb.GetInterfaceResponse, error) {
	iface, status := s.SwitchAgent.GetInterface(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name: request.GetInterfaceName(),
	})
	if status != nil {
		return &pb.GetInterfaceResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: fmt.Sprintf("failed to get interface: %v", status.Message),
			},
		}, nil
	}

	return &pb.GetInterfaceResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
		Interface: &pb.Interface{
			Name:              iface.Name,
			MacAddress:        iface.MacAddress,
			OperationalStatus: string(iface.OperationStatus),
			AdminStatus:       string(iface.AdminStatus),
		},
	}, nil
}

func (s *proxyServer) GetInterfaceNeighbor(ctx context.Context, request *pb.GetInterfaceNeighborRequest) (*pb.GetInterfaceNeighborResponse, error) {
	ifaceNeighbor, status := s.SwitchAgent.GetInterfaceNeighbor(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name: request.GetInterfaceName(),
	})
	if status != nil {
		return &pb.GetInterfaceNeighborResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: fmt.Sprintf("failed to get interface neighbor: %v", status.Message),
			},
		}, nil
	}

	return &pb.GetInterfaceNeighborResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
		Interface: request.GetInterfaceName(),
		Neighbor: &pb.InterfaceNeighbor{
			MacAddress:            ifaceNeighbor.MacAddress,
			NeighborInterfaceName: ifaceNeighbor.Handle,
			SystemName:            ifaceNeighbor.SystemName,
		},
	}, nil
}

func StartServer() {
	flag.Parse()

	lis, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	swAgent, err := sonic.NewSonicRedisAgent(*redisAddr)
	if err != nil {
		log.Fatalf("failed to create SonicRedisAgent: %v", err)
		panic(err)
	}

	pb.RegisterSwitchAgentServiceServer(s, &proxyServer{
		SwitchAgent: swAgent,
	})

	// Register reflection service on gRPC server for debugging
	reflection.Register(s)

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

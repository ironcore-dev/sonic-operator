// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package agent_server

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	switchAgent "github.com/ironcore-dev/sonic-operator/internal/agent/interface"
	"github.com/ironcore-dev/sonic-operator/internal/agent/metrics"
	pb "github.com/ironcore-dev/sonic-operator/internal/agent/proto"
	"github.com/ironcore-dev/sonic-operator/internal/agent/sonic"
	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port          = flag.Int("port", 50051, "The server port")
	redisAddr     = flag.String("redis-addr", "127.0.0.1:6379", "The Redis address")
	metricsPort   = flag.Int("metrics-port", 9100, "The metrics server port")
	metricsConfig = flag.String("metrics-config", "", "Path to metrics mapping config YAML (uses built-in defaults if empty)")
)

type proxyServer struct {
	pb.UnimplementedSwitchAgentServiceServer

	SwitchAgent switchAgent.SwitchAgent
}

func (s *proxyServer) GetDeviceInfo(ctx context.Context, request *pb.GetDeviceInfoRequest) (*pb.GetDeviceInfoResponse, error) {
	log.Printf("GetDeviceInfo called")

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
	log.Printf("ListInterfaces called")

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
			NativeName:        iface.NativeName,
			AliasName:         iface.AliasName,
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
	log.Printf("SetInterfaceAdminStatus called: interface=%s, status=%s", request.GetInterfaceName(), request.GetAdminStatus())

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
	log.Printf("ListPorts called")

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
	log.Printf("GetInterface called: interface=%s", request.GetInterfaceName())

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
			NativeName:        iface.NativeName,
			AliasName:         iface.AliasName,
			MacAddress:        iface.MacAddress,
			OperationalStatus: string(iface.OperationStatus),
			AdminStatus:       string(iface.AdminStatus),
		},
	}, nil
}

func (s *proxyServer) SetInterfaceAliasName(ctx context.Context, request *pb.SetInterfaceAliasNameRequest) (*pb.SetInterfaceAliasNameResponse, error) {
	log.Printf("SetInterfaceAliasName called: interface=%s, alias=%s", request.GetInterfaceName(), request.GetAliasName())

	iface, status := s.SwitchAgent.SetInterfaceAliasName(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name:      request.GetInterfaceName(),
		AliasName: request.GetAliasName(),
	})

	if status != nil {
		return &pb.SetInterfaceAliasNameResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: status.Message,
			},
		}, nil
	}

	return &pb.SetInterfaceAliasNameResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
		Interface: &pb.Interface{
			Name:              iface.Name,
			AliasName:         iface.AliasName,
			NativeName:        iface.GetNativeName(),
			MacAddress:        "",
			OperationalStatus: string(iface.OperationStatus),
			AdminStatus:       string(iface.AdminStatus),
		},
	}, nil
}

func (s *proxyServer) GetInterfaceNeighbor(ctx context.Context, request *pb.GetInterfaceNeighborRequest) (*pb.GetInterfaceNeighborResponse, error) {
	log.Printf("GetInterfaceNeighbor called: interface=%s", request.GetInterfaceName())

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

func (s *proxyServer) SaveConfig(ctx context.Context, request *pb.SaveConfigRequest) (*pb.SaveConfigResponse, error) {
	log.Printf("SaveConfig called")

	status := s.SwitchAgent.SaveConfig(ctx)
	if status != nil {
		return &pb.SaveConfigResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: fmt.Sprintf("failed to save config: %v", status.Message),
			},
		}, nil
	}

	return &pb.SaveConfigResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
	}, nil
}

// NewProxyServer creates a proxyServer backed by the given SwitchAgent.
// This is exported so tests can instantiate a server with a fake agent.
func NewProxyServer(switchAgentImpl switchAgent.SwitchAgent) pb.SwitchAgentServiceServer {
	return &proxyServer{SwitchAgent: switchAgentImpl}
}

func StartServer() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	swAgent, err := sonic.NewSonicRedisAgent(*redisAddr)
	if err != nil {
		log.Fatalf("failed to create SonicRedisAgent: %v", err)
		panic(err)
	}

	// Start Prometheus metrics HTTP server
	metricsSrv := metrics.NewMetricsServer(fmt.Sprintf(":%d", *metricsPort), swAgent, sonic.GetSonicVersionInfo, *metricsConfig)
	go func() {
		log.Printf("metrics server listening at :%d", *metricsPort)
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("metrics server failed: %v", err)
		}
	}()

	pb.RegisterSwitchAgentServiceServer(s, NewProxyServer(swAgent))

	// Register reflection service on gRPC server for debugging
	reflection.Register(s)

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down", sig)

		// Gracefully stop the gRPC server (drains in-flight RPCs)
		s.GracefulStop()

		// Shut down the metrics HTTP server with a timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsSrv.Shutdown(ctx); err != nil {
			log.Printf("metrics server shutdown error: %v", err)
		}
	}()

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

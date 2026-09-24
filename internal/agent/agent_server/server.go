// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package agent_server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"time"

	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"
	pb "github.com/ironcore-dev/sonic-operator/pkg/agent/proto"

	switchAgent "github.com/ironcore-dev/sonic-operator/internal/agent/interface"
	"github.com/ironcore-dev/sonic-operator/internal/agent/sonic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

var (
	port        = flag.Int("port", 50051, "The server port")
	redisAddr   = flag.String("redis-addr", "127.0.0.1:6379", "The Redis address")
	tlsCertFile = flag.String("tls-cert-file", "", "Path to PEM server certificate (optional, auto-generated if omitted).")
	tlsKeyFile  = flag.String("tls-key-file", "", "Path to PEM private key (optional, auto-generated if omitted).")
	enablePlain = flag.Bool("enable-plain-mode", false, "Disable TLS and run in plaintext mode.")
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
			Name:              iface.NativeName,
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
		NativeName:  request.GetInterfaceName(),
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
			Name:              iface.NativeName,
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
		NativeName: request.GetInterfaceName(),
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
			Name:              iface.NativeName,
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
		NativeName: request.GetInterfaceName(),
		AliasName:  request.GetAliasName(),
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
			Name:              iface.NativeName,
			AliasName:         iface.AliasName,
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
		NativeName: request.GetInterfaceName(),
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
				Message: status.Message,
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

func (s *proxyServer) Reboot(ctx context.Context, request *pb.RebootRequest) (*pb.RebootResponse, error) {
	log.Printf("Reboot called")

	status := s.SwitchAgent.Reboot(ctx)
	if status != nil {
		return &pb.RebootResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: status.Message,
			},
		}, nil
	}

	return &pb.RebootResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
	}, nil
}

func (s *proxyServer) OnieBootModeInstall(ctx context.Context, request *pb.OnieBootModeInstallRequest) (*pb.OnieBootModeInstallResponse, error) {
	log.Printf("OnieBootModeInstall called")

	status := s.SwitchAgent.OnieBootModeInstall(ctx)
	if status != nil {
		return &pb.OnieBootModeInstallResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: status.Message,
			},
		}, nil
	}

	return &pb.OnieBootModeInstallResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
	}, nil
}

func (s *proxyServer) RestartSystemdService(ctx context.Context, request *pb.RestartSystemdServiceRequest) (*pb.RestartSystemdServiceResponse, error) {
	log.Printf("RestartSystemdService called: service=%s", request.GetServiceName())

	status := s.SwitchAgent.RestartSystemdService(ctx, request.GetServiceName())
	if status != nil {
		return &pb.RestartSystemdServiceResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: status.Message,
			},
		}, nil
	}

	return &pb.RestartSystemdServiceResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
	}, nil
}

func (s *proxyServer) RebootCause(ctx context.Context, _ *pb.RebootCauseRequest) (*pb.RebootCauseResponse, error) {
	log.Printf("RebootCause called")

	cause, status := s.SwitchAgent.RebootCause(ctx)
	if status != nil {
		return &pb.RebootCauseResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: status.Message,
			},
		}, nil
	}

	return &pb.RebootCauseResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
		Cause: cause,
	}, nil
}

func (s *proxyServer) FactoryReset(ctx context.Context, _ *pb.FactoryResetRequest) (*pb.FactoryResetResponse, error) {
	log.Printf("FactoryReset called")

	status := s.SwitchAgent.FactoryReset(ctx)
	if status != nil {
		return &pb.FactoryResetResponse{
			Status: &pb.Status{
				Code:    status.Code,
				Message: status.Message,
			},
		}, nil
	}

	return &pb.FactoryResetResponse{
		Status: &pb.Status{
			Code:    0,
			Message: "Success",
		},
	}, nil
}

func (s *proxyServer) GetReadiness(ctx context.Context, _ *pb.GetReadinessRequest) (*pb.GetReadinessResponse, error) {
	log.Printf("GetReadiness called")

	ready, status := s.SwitchAgent.GetReadiness(ctx)
	if status != nil {
		return &pb.GetReadinessResponse{
			Status: &pb.Status{Code: status.Code, Message: status.Message},
		}, nil
	}

	return &pb.GetReadinessResponse{Ready: ready}, nil
}

func (s *proxyServer) Reprovision(ctx context.Context, _ *pb.ReprovisionRequest) (*pb.ReprovisionResponse, error) {
	log.Printf("Reprovision called")

	agentStatus := s.SwitchAgent.Reprovision(ctx)
	if agentStatus != nil {
		return &pb.ReprovisionResponse{
			Status: &pb.Status{Code: agentStatus.Code, Message: agentStatus.Message},
		}, nil
	}
	return &pb.ReprovisionResponse{}, nil
}

// NewProxyServer creates a proxyServer backed by the given SwitchAgent.
// This is exported so tests can instantiate a server with a fake agent.
func NewProxyServer(switchAgentImpl switchAgent.SwitchAgent) pb.SwitchAgentServiceServer {
	return &proxyServer{SwitchAgent: switchAgentImpl}
}

func generateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate key: %w", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "sonic-agent"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshal key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return tls.X509KeyPair(certPEM, keyPEM)
}

func buildServerOptions(certFile, keyFile string, plainMode bool) ([]grpc.ServerOption, error) {
	if plainMode {
		if certFile != "" || keyFile != "" {
			return nil, fmt.Errorf("--enable-plain-mode conflicts with --tls-cert-file / --tls-key-file")
		}
		return nil, nil
	}

	var (
		cert tls.Certificate
		err  error
	)
	if certFile == "" && keyFile == "" {
		log.Printf("No TLS cert/key provided; generating self-signed certificate")
		cert, err = generateSelfSignedCert()
		if err != nil {
			return nil, fmt.Errorf("failed to generate self-signed cert: %w", err)
		}
	} else if certFile != "" && keyFile != "" {
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair: %w", err)
		}
	} else {
		return nil, fmt.Errorf("both --tls-cert-file and --tls-key-file must be set together (got cert=%q, key=%q)", certFile, keyFile)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
		MinVersion:   tls.VersionTLS12,
	}
	return []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}, nil
}

func StartServer() {
	flag.Parse()

	lis, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serverOpts, err := buildServerOptions(*tlsCertFile, *tlsKeyFile, *enablePlain)
	if err != nil {
		log.Fatalf("invalid server configuration: %v", err)
	}
	var s *grpc.Server
	if serverOpts != nil {
		s = grpc.NewServer(serverOpts...)
		log.Printf("gRPC server starting with TLS")
	} else {
		s = grpc.NewServer()
		log.Printf("gRPC server starting in plaintext mode")
	}

	swAgent, err := sonic.NewSonicRedisAgent(*redisAddr)
	if err != nil {
		log.Fatalf("failed to create SonicRedisAgent: %v", err)
	}

	pb.RegisterSwitchAgentServiceServer(s, NewProxyServer(swAgent))
	pb.RegisterFabricSonicServiceServer(s, NewFabricSonicServiceServer(swAgent))

	// Register reflection service on gRPC server for debugging
	reflection.Register(s)

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

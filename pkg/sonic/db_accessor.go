// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// PortAccessor covers all port-level read and write operations.
type PortAccessor interface {
	ListPortNames(ctx context.Context) ([]string, error)
	GetPortSpeed(ctx context.Context, portName string) (int, error)
	GetPortSupportedSpeeds(ctx context.Context, portName string) (string, error)
	GetTransceiverType(ctx context.Context, portName string) (string, error)
	GetPortAlias(ctx context.Context, portName string) (string, error)
	GetLLDPEntry(ctx context.Context, portName string) (map[string]string, error)
	ListPortTableEntries(ctx context.Context) (map[string]map[string]string, error)
	HasPort(ctx context.Context, portName string) (bool, error)
	SetMTU(ctx context.Context, portName string, mtu int) error
	SetFEC(ctx context.Context, portName string, fec string) error
	SetSpeed(ctx context.Context, portName string, speedMbps int) error
	SetPortAlias(ctx context.Context, portName, alias string) error
}

// InterfaceAccessor covers loopback management, IP address operations, and interface status reads.
type InterfaceAccessor interface {
	GetAdminStatus(ctx context.Context, interfaceName string) (bool, error)
	GetOperStatus(ctx context.Context, interfaceName string) (bool, error)
	SetAdminStatus(ctx context.Context, interfaceName string, adminStatus string) error
	EnsureLoopback(ctx context.Context, loopbackName string) error
	DeleteLoopback(ctx context.Context, loopbackName string) error
	AddIPAddresses(ctx context.Context, interfaceName string, prefixes []string) error
	RemoveIPAddresses(ctx context.Context, interfaceName string, prefixes []string) error
	ListIPAddresses(ctx context.Context, interfaceName string) ([]string, error)
	SyncIPAddresses(ctx context.Context, interfaceName string, desired []string) error
}

// VLANAccessor covers VLAN lifecycle, membership, and DHCP relay operations.
type VLANAccessor interface {
	EnsureVLAN(ctx context.Context, vlanName string) error
	EnsureVLANInterface(ctx context.Context, vlanName, vrfName string) error
	DeleteVLAN(ctx context.Context, vlanName string) error
	DeleteVLANInterface(ctx context.Context, vlanName string) error
	HasVLAN(ctx context.Context, vlanName string) (bool, error)
	SetVLANDescription(ctx context.Context, vlanName, description string) error
	EnsureVLANMember(ctx context.Context, vlanName, portName, taggingMode string) error
	RemoveVLANMember(ctx context.Context, vlanName, portName string) error
	ListPortVLANs(ctx context.Context, portName string) ([]string, error)
	SyncPortVLANs(ctx context.Context, portName string, desired map[string]string) error
	EnsureDHCPRelay(ctx context.Context, vlanName string, servers []string) error
	DeleteDHCPRelay(ctx context.Context, vlanName string) error
	HasDHCPRelay(ctx context.Context, vlanName string) (bool, error)
}

// DeviceAccessor covers standard SONiC switch-level metadata.
type DeviceAccessor interface {
	SetHostname(ctx context.Context, hostname string) error
	GetDeviceMetadata(ctx context.Context) (map[string]string, error)
}

// DBAccessor is the full SONiC Redis accessor composed from all domain sub-interfaces.
// NewDBAccessor returns a DBAccessor; consumers that only need a sub-domain
// may accept the narrower interface directly.
type DBAccessor interface {
	PortAccessor
	InterfaceAccessor
	VLANAccessor
	DeviceAccessor
}

type dbAccessor struct {
	configDB *redis.Client
	stateDB  *redis.Client
	applDB   *redis.Client
}

// NewDBAccessor returns a DBAccessor backed by the provided Redis clients.
// The caller is responsible for connecting and closing the clients.
func NewDBAccessor(configDB, stateDB, applDB *redis.Client) DBAccessor {
	return &dbAccessor{configDB: configDB, stateDB: stateDB, applDB: applDB}
}

// compile-time interface satisfaction checks
var _ PortAccessor = (*dbAccessor)(nil)
var _ InterfaceAccessor = (*dbAccessor)(nil)
var _ VLANAccessor = (*dbAccessor)(nil)
var _ DeviceAccessor = (*dbAccessor)(nil)
var _ DBAccessor = (*dbAccessor)(nil)

func scanKeys(ctx context.Context, client *redis.Client, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		batch, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func validatePortName(portName string) error {
	if strings.HasPrefix(portName, "Ethernet") {
		return nil
	}
	return fmt.Errorf("invalid port name %q: must start with Ethernet", portName)
}

func validateVlanName(vlanName string) error {
	if !strings.HasPrefix(vlanName, "Vlan") {
		return fmt.Errorf("invalid VLAN name %q: must start with Vlan", vlanName)
	}
	id, err := strconv.Atoi(strings.TrimPrefix(vlanName, "Vlan"))
	if err != nil || id < 1 || id > 4094 {
		return fmt.Errorf("invalid VLAN name %q: numeric suffix must be between 1 and 4094", vlanName)
	}
	return nil
}

func validateLoopbackName(loopbackName string) error {
	if !strings.HasPrefix(loopbackName, "Loopback") {
		return fmt.Errorf("invalid loopback name %q: must start with Loopback", loopbackName)
	}
	return nil
}

// ParseSupportedSpeeds converts a comma-separated Mbps string to a Gbps slice (values divided by 1000),
// falling back to a single entry derived from fallbackMbps when raw is empty.
func ParseSupportedSpeeds(raw string, fallbackMbps int) []int32 {
	if raw != "" {
		var out []int32
		for _, s := range strings.Split(raw, ",") {
			if mbps, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && mbps > 0 {
				out = append(out, int32(mbps/1000))
			}
		}
		return out
	}
	if fallbackMbps > 0 {
		return []int32{int32(fallbackMbps / 1000)}
	}
	return nil
}

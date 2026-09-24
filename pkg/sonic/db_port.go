// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func (db *dbAccessor) ListPortNames(ctx context.Context) ([]string, error) {
	keys, err := scanKeys(ctx, db.configDB, "PORT|*")
	if err != nil {
		return nil, fmt.Errorf("failed to list PORT keys: %w", err)
	}
	names := make([]string, len(keys))
	for i, k := range keys {
		names[i] = strings.TrimPrefix(k, "PORT|")
	}
	return names, nil
}

func (db *dbAccessor) GetPortConfig(ctx context.Context, portName string) (*PortConfig, error) {
	fields, err := db.configDB.HGetAll(ctx, "PORT|"+portName).Result()
	if err != nil {
		return nil, err
	}
	speed, _ := strconv.Atoi(fields["speed"])
	return &PortConfig{Speed: speed}, nil
}

func (db *dbAccessor) GetPortState(ctx context.Context, portName string) *PortState {
	raw, err := db.stateDB.HGet(ctx, "PORT_TABLE|"+portName, "supported_speeds").Result()
	if err != nil {
		return &PortState{} // field absent on virtual/copper ports
	}
	return &PortState{SupportedSpeeds: raw}
}

func (db *dbAccessor) GetTransceiverInfo(ctx context.Context, portName string) *TransceiverInfo {
	t, err := db.stateDB.HGet(ctx, "TRANSCEIVER_INFO|"+portName, "type").Result()
	if err != nil {
		return &TransceiverInfo{} // absent on virtual/copper ports
	}
	return &TransceiverInfo{Type: t}
}

func (db *dbAccessor) GetAdminStatus(ctx context.Context, interfaceName string) (bool, error) {
	if strings.HasPrefix(interfaceName, "Loopback") {
		val, err := db.configDB.HGet(ctx, "LOOPBACK_INTERFACE|"+interfaceName, "admin_status").Result()
		if err != nil {
			return false, nil
		}
		return val == "up", nil
	}
	if strings.HasPrefix(interfaceName, "Vlan") {
		val, err := db.applDB.HGet(ctx, "VLAN_TABLE:"+interfaceName, "admin_status").Result()
		if err != nil {
			return false, nil
		}
		return val == "up", nil
	}
	if strings.HasPrefix(interfaceName, "Ethernet") {
		val, err := db.applDB.HGet(ctx, "PORT_TABLE:"+interfaceName, "admin_status").Result()
		if err != nil {
			return false, nil
		}
		return val == "up", nil
	}
	return false, fmt.Errorf("unknown interface: %s", interfaceName)
}

func (db *dbAccessor) GetOperStatus(ctx context.Context, interfaceName string) (bool, error) {
	if strings.HasPrefix(interfaceName, "Loopback") {
		val, err := db.configDB.HGet(ctx, "LOOPBACK_INTERFACE|"+interfaceName, "admin_status").Result()
		if err != nil {
			return false, nil // absent means down
		}
		return val == "up", nil
	}
	if strings.HasPrefix(interfaceName, "Vlan") {
		val, err := db.applDB.HGet(ctx, "VLAN_TABLE:"+interfaceName, "admin_status").Result()
		if err != nil {
			return false, nil // absent means down
		}
		return val == "up", nil
	}
	if strings.HasPrefix(interfaceName, "Ethernet") {
		val, err := db.applDB.HGet(ctx, "PORT_TABLE:"+interfaceName, "oper_status").Result()
		if err != nil {
			return false, nil // absent means down
		}
		return val == "up", nil
	}
	return false, fmt.Errorf("unknown interface: %s", interfaceName)
}

func (db *dbAccessor) GetPortFields(ctx context.Context, portName string) (map[string]string, error) {
	fields, err := db.configDB.HGetAll(ctx, "PORT|"+portName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get port fields for %s: %w", portName, err)
	}
	return fields, nil
}

func (db *dbAccessor) GetPortStateFields(ctx context.Context, portName string) (map[string]string, error) {
	fields, err := db.stateDB.HGetAll(ctx, "PORT_TABLE|"+portName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get port state fields for %s: %w", portName, err)
	}
	return fields, nil
}

func (db *dbAccessor) GetPortApplFields(ctx context.Context, portName string) (map[string]string, error) {
	fields, err := db.applDB.HGetAll(ctx, "PORT_TABLE:"+portName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get port appl fields for %s: %w", portName, err)
	}
	return fields, nil
}

func (db *dbAccessor) GetPortAlias(ctx context.Context, portName string) (string, error) {
	alias, err := db.configDB.HGet(ctx, "PORT|"+portName, "alias").Result()
	if err != nil {
		return "", fmt.Errorf("failed to get alias for port %s: %w", portName, err)
	}
	return alias, nil
}

// GetLLDPEntry returns the LLDP_ENTRY_TABLE fields for portName from APPL_DB.
// The bool is false when the entry does not exist.
func (db *dbAccessor) GetLLDPEntry(ctx context.Context, portName string) (map[string]string, bool, error) {
	key := "LLDP_ENTRY_TABLE:" + portName
	n, err := db.applDB.Exists(ctx, key).Result()
	if err != nil {
		return nil, false, fmt.Errorf("failed to check LLDP entry for %s: %w", portName, err)
	}
	if n == 0 {
		return nil, false, nil
	}
	fields, err := db.applDB.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get LLDP entry for %s: %w", portName, err)
	}
	return fields, true, nil
}

// ListPortTableEntries returns all PORT_TABLE:* entries from APPL_DB as a map of portName → fields.
func (db *dbAccessor) ListPortTableEntries(ctx context.Context) (map[string]map[string]string, error) {
	keys, err := scanKeys(ctx, db.applDB, "PORT_TABLE:*")
	if err != nil {
		return nil, fmt.Errorf("failed to list PORT_TABLE keys: %w", err)
	}
	result := make(map[string]map[string]string, len(keys))
	for _, key := range keys {
		portName := strings.TrimPrefix(key, "PORT_TABLE:")
		fields, err := db.applDB.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get PORT_TABLE fields for %s: %w", portName, err)
		}
		result[portName] = fields
	}
	return result, nil
}

func (db *dbAccessor) HasPort(ctx context.Context, portName string) (bool, error) {
	n, err := db.configDB.Exists(ctx, "PORT|"+portName).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of port %s: %w", portName, err)
	}
	return n > 0, nil
}

func (db *dbAccessor) SetMTU(ctx context.Context, interfaceName string, mtu int) error {
	if strings.HasPrefix(interfaceName, "Loopback") {
		return fmt.Errorf("MTU cannot be set on loopback interface %s", interfaceName)
	}
	if err := db.configDB.HSet(ctx, "PORT|"+interfaceName, "mtu", strconv.Itoa(mtu)).Err(); err != nil {
		return fmt.Errorf("failed to set MTU for %s: %w", interfaceName, err)
	}
	return nil
}

func (db *dbAccessor) SetFEC(ctx context.Context, interfaceName string, fec string) error {
	switch fec {
	case "rs", "fc", "none":
	default:
		return fmt.Errorf("invalid FEC value %q for %s: must be rs, fc, or none", fec, interfaceName)
	}
	if err := db.configDB.HSet(ctx, "PORT|"+interfaceName, "fec", fec).Err(); err != nil {
		return fmt.Errorf("failed to set FEC for %s: %w", interfaceName, err)
	}
	return nil
}

func (db *dbAccessor) SetSpeed(ctx context.Context, interfaceName string, speedMbps int) error {
	if speedMbps <= 0 {
		return fmt.Errorf("invalid speed %d for %s: must be positive", speedMbps, interfaceName)
	}
	if err := db.configDB.HSet(ctx, "PORT|"+interfaceName, "speed", strconv.Itoa(speedMbps)).Err(); err != nil {
		return fmt.Errorf("failed to set speed for %s: %w", interfaceName, err)
	}
	return nil
}

func (db *dbAccessor) SetAdminState(ctx context.Context, interfaceName string, adminStatus string) error {
	var key string
	if strings.HasPrefix(interfaceName, "Loopback") {
		key = "LOOPBACK_INTERFACE|" + interfaceName
	} else if strings.HasPrefix(interfaceName, "Vlan") {
		key = "VLAN|" + interfaceName
	} else {
		key = "PORT|" + interfaceName
	}
	if err := db.configDB.HSet(ctx, key, "admin_status", adminStatus).Err(); err != nil {
		return fmt.Errorf("failed to set admin state for %s: %w", interfaceName, err)
	}
	return nil
}

func (db *dbAccessor) SetPortAdminStatus(ctx context.Context, portName, status string) error {
	if err := db.configDB.HSet(ctx, "PORT|"+portName, "admin_status", status).Err(); err != nil {
		return fmt.Errorf("failed to set admin_status for port %s: %w", portName, err)
	}
	return nil
}

func (db *dbAccessor) SetPortAlias(ctx context.Context, portName, alias string) error {
	if err := db.configDB.HSet(ctx, "PORT|"+portName, "alias", alias).Err(); err != nil {
		return fmt.Errorf("failed to set alias for port %s: %w", portName, err)
	}
	return nil
}

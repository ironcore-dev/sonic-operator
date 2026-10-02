// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
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

func (db *dbAccessor) GetPortSpeed(ctx context.Context, portName string) (int, error) {
	if err := validatePortName(portName); err != nil {
		return 0, err
	}
	raw, err := db.configDB.HGet(ctx, "PORT|"+portName, "speed").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get speed for port %s: %w", portName, err)
	}
	speed, _ := strconv.Atoi(raw)
	return speed, nil
}

func (db *dbAccessor) GetPortSupportedSpeeds(ctx context.Context, portName string) (string, error) {
	if err := validatePortName(portName); err != nil {
		return "", err
	}
	raw, err := db.stateDB.HGet(ctx, "PORT_TABLE|"+portName, "supported_speeds").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil // field absent on virtual/copper ports
		}
		return "", fmt.Errorf("failed to get supported speeds for port %s: %w", portName, err)
	}
	return raw, nil
}

func (db *dbAccessor) GetTransceiverType(ctx context.Context, portName string) (string, error) {
	if err := validatePortName(portName); err != nil {
		return "", err
	}
	t, err := db.stateDB.HGet(ctx, "TRANSCEIVER_INFO|"+portName, "type").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil // absent on virtual/copper ports
		}
		return "", fmt.Errorf("failed to get transceiver type for %s: %w", portName, err)
	}
	return t, nil
}

func (db *dbAccessor) GetPortAlias(ctx context.Context, portName string) (string, error) {
	if err := validatePortName(portName); err != nil {
		return "", err
	}
	alias, err := db.configDB.HGet(ctx, "PORT|"+portName, "alias").Result()
	if err != nil {
		return "", fmt.Errorf("failed to get alias for port %s: %w", portName, err)
	}
	return alias, nil
}

// GetLLDPEntry returns the LLDP_ENTRY_TABLE fields for portName from APPL_DB.
// Returns nil map when the entry does not exist.
func (db *dbAccessor) GetLLDPEntry(ctx context.Context, portName string) (map[string]string, error) {
	if err := validatePortName(portName); err != nil {
		return nil, err
	}
	key := "LLDP_ENTRY_TABLE:" + portName
	fields, err := db.applDB.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLDP entry for %s: %w", portName, err)
	}
	if len(fields) == 0 {
		return nil, nil
	}
	return fields, nil
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
	if err := validatePortName(portName); err != nil {
		return false, err
	}
	n, err := db.configDB.Exists(ctx, "PORT|"+portName).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of port %s: %w", portName, err)
	}
	return n > 0, nil
}

func (db *dbAccessor) SetMTU(ctx context.Context, portName string, mtu int) error {
	if err := validatePortName(portName); err != nil {
		return err
	}
	if mtu < 68 || mtu > 9216 {
		return fmt.Errorf("invalid MTU %d for %s: must be between 68 and 9216", mtu, portName)
	}
	if err := db.configDB.HSet(ctx, "PORT|"+portName, "mtu", strconv.Itoa(mtu)).Err(); err != nil {
		return fmt.Errorf("failed to set MTU for %s: %w", portName, err)
	}
	return nil
}

func (db *dbAccessor) SetFEC(ctx context.Context, portName string, fec string) error {
	if err := validatePortName(portName); err != nil {
		return err
	}
	switch fec {
	case "rs", "fc", "none":
	default:
		return fmt.Errorf("invalid FEC value %q for %s: must be rs, fc, or none", fec, portName)
	}
	if err := db.configDB.HSet(ctx, "PORT|"+portName, "fec", fec).Err(); err != nil {
		return fmt.Errorf("failed to set FEC for %s: %w", portName, err)
	}
	return nil
}

func (db *dbAccessor) SetSpeed(ctx context.Context, portName string, speedMbps int) error {
	if err := validatePortName(portName); err != nil {
		return err
	}
	if speedMbps <= 0 {
		return fmt.Errorf("invalid speed %d for %s: must be positive", speedMbps, portName)
	}
	if err := db.configDB.HSet(ctx, "PORT|"+portName, "speed", strconv.Itoa(speedMbps)).Err(); err != nil {
		return fmt.Errorf("failed to set speed for %s: %w", portName, err)
	}
	return nil
}

func (db *dbAccessor) SetPortAlias(ctx context.Context, portName, alias string) error {
	if err := validatePortName(portName); err != nil {
		return err
	}
	if alias == "" {
		return fmt.Errorf("alias must not be empty for port %s", portName)
	}
	if err := db.configDB.HSet(ctx, "PORT|"+portName, "alias", alias).Err(); err != nil {
		return fmt.Errorf("failed to set alias for port %s: %w", portName, err)
	}
	return nil
}

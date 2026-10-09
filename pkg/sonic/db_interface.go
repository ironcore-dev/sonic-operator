// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/redis/go-redis/v9"
)

func hgetIsUp(ctx context.Context, client *redis.Client, key, field string) (bool, error) {
	val, err := client.HGet(ctx, key, field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return val == "up", nil
}

func (db *dbAccessor) GetAdminStatus(ctx context.Context, interfaceName string) (bool, error) {
	switch {
	case strings.HasPrefix(interfaceName, "Loopback"):
		return hgetIsUp(ctx, db.configDB, "LOOPBACK_INTERFACE|"+interfaceName, "admin_status")
	case strings.HasPrefix(interfaceName, "Vlan"):
		return hgetIsUp(ctx, db.applDB, "VLAN_TABLE:"+interfaceName, "admin_status")
	case strings.HasPrefix(interfaceName, "Ethernet"):
		return hgetIsUp(ctx, db.applDB, "PORT_TABLE:"+interfaceName, "admin_status")
	}
	return false, fmt.Errorf("unknown interface: %s", interfaceName)
}

func (db *dbAccessor) GetOperStatus(ctx context.Context, interfaceName string) (bool, error) {
	switch {
	case strings.HasPrefix(interfaceName, "Loopback"):
		// SONiC does not publish a separate oper_status for loopbacks; mirrors admin_status.
		return hgetIsUp(ctx, db.configDB, "LOOPBACK_INTERFACE|"+interfaceName, "admin_status")
	case strings.HasPrefix(interfaceName, "Vlan"):
		// SONiC does not publish a separate oper_status for VLANs; mirrors admin_status.
		return hgetIsUp(ctx, db.applDB, "VLAN_TABLE:"+interfaceName, "admin_status")
	case strings.HasPrefix(interfaceName, "Ethernet"):
		return hgetIsUp(ctx, db.applDB, "PORT_TABLE:"+interfaceName, "oper_status")
	}
	return false, fmt.Errorf("unknown interface: %s", interfaceName)
}

func (db *dbAccessor) SetAdminStatus(ctx context.Context, interfaceName string, adminStatus string) error {
	if adminStatus != "up" && adminStatus != "down" {
		return fmt.Errorf("invalid admin_status %q for %s: must be \"up\" or \"down\"", adminStatus, interfaceName)
	}
	var key string
	if strings.HasPrefix(interfaceName, "Loopback") {
		key = "LOOPBACK_INTERFACE|" + interfaceName
	} else if strings.HasPrefix(interfaceName, "Vlan") {
		key = "VLAN|" + interfaceName
	} else if strings.HasPrefix(interfaceName, "Ethernet") {
		key = "PORT|" + interfaceName
	} else if strings.HasPrefix(interfaceName, "PortChannel") {
		key = "PORTCHANNEL|" + interfaceName
	} else {
		return fmt.Errorf("unknown interface type for admin status: %s", interfaceName)
	}
	if err := db.configDB.HSet(ctx, key, "admin_status", adminStatus).Err(); err != nil {
		return fmt.Errorf("failed to set admin status for %s: %w", interfaceName, err)
	}
	return nil
}

func interfaceIPTable(interfaceName string) (string, error) {
	switch {
	case strings.HasPrefix(interfaceName, "Loopback"):
		return "LOOPBACK_INTERFACE", nil
	case strings.HasPrefix(interfaceName, "Vlan"):
		return "VLAN_INTERFACE", nil
	case strings.HasPrefix(interfaceName, "Ethernet"):
		return "INTERFACE", nil
	case strings.HasPrefix(interfaceName, "PortChannel"):
		return "PORTCHANNEL_INTERFACE", nil
	default:
		return "", fmt.Errorf("unknown interface type for IP assignment: %s", interfaceName)
	}
}

func (db *dbAccessor) EnsureLoopback(ctx context.Context, loopbackName string) error {
	if err := validateLoopbackName(loopbackName); err != nil {
		return err
	}
	if err := db.configDB.HSet(ctx, "LOOPBACK_INTERFACE|"+loopbackName, "NULL", "NULL").Err(); err != nil {
		return fmt.Errorf("failed to ensure loopback interface %s: %w", loopbackName, err)
	}
	return nil
}

func (db *dbAccessor) DeleteLoopback(ctx context.Context, loopbackName string) error {
	if err := validateLoopbackName(loopbackName); err != nil {
		return err
	}
	if err := db.configDB.Del(ctx, "LOOPBACK_INTERFACE|"+loopbackName).Err(); err != nil {
		return fmt.Errorf("failed to delete loopback interface %s: %w", loopbackName, err)
	}
	return nil
}

func (db *dbAccessor) AddIPAddresses(ctx context.Context, interfaceName string, prefixes []string) error {
	table, err := interfaceIPTable(interfaceName)
	if err != nil {
		return err
	}
	for _, prefix := range prefixes {
		if _, err := netip.ParsePrefix(prefix); err != nil {
			return fmt.Errorf("invalid IP prefix %q: %w", prefix, err)
		}
		if err := db.configDB.HSet(ctx, table+"|"+interfaceName+"|"+prefix, "NULL", "NULL").Err(); err != nil {
			return fmt.Errorf("failed to add IP %s to %s: %w", prefix, interfaceName, err)
		}
	}
	return nil
}

func (db *dbAccessor) RemoveIPAddresses(ctx context.Context, interfaceName string, prefixes []string) error {
	table, err := interfaceIPTable(interfaceName)
	if err != nil {
		return err
	}
	for _, prefix := range prefixes {
		if _, err := netip.ParsePrefix(prefix); err != nil {
			return fmt.Errorf("invalid IP prefix %q: %w", prefix, err)
		}
		if err := db.configDB.Del(ctx, table+"|"+interfaceName+"|"+prefix).Err(); err != nil {
			return fmt.Errorf("failed to remove IP %s from %s: %w", prefix, interfaceName, err)
		}
	}
	return nil
}

func (db *dbAccessor) ListIPAddresses(ctx context.Context, interfaceName string) ([]string, error) {
	table, err := interfaceIPTable(interfaceName)
	if err != nil {
		return nil, err
	}
	keys, err := scanKeys(ctx, db.configDB, table+"|"+interfaceName+"|*")
	if err != nil {
		return nil, fmt.Errorf("failed to list IP addresses for %s: %w", interfaceName, err)
	}
	pfx := table + "|" + interfaceName + "|"
	addrs := make([]string, 0, len(keys))
	for _, k := range keys {
		addrs = append(addrs, strings.TrimPrefix(k, pfx))
	}
	return addrs, nil
}

func (db *dbAccessor) SyncIPAddresses(ctx context.Context, interfaceName string, desired []string) error {
	current, err := db.ListIPAddresses(ctx, interfaceName)
	if err != nil {
		return err
	}

	desiredSet := make(map[string]struct{}, len(desired))
	for _, p := range desired {
		desiredSet[p] = struct{}{}
	}

	var toRemove []string
	for _, p := range current {
		if _, ok := desiredSet[p]; !ok {
			toRemove = append(toRemove, p)
		}
	}
	if len(toRemove) > 0 {
		if err := db.RemoveIPAddresses(ctx, interfaceName, toRemove); err != nil {
			return err
		}
	}

	currentSet := make(map[string]struct{}, len(current))
	for _, p := range current {
		currentSet[p] = struct{}{}
	}
	var toAdd []string
	for _, p := range desired {
		if _, ok := currentSet[p]; !ok {
			toAdd = append(toAdd, p)
		}
	}
	if len(toAdd) > 0 {
		if err := db.AddIPAddresses(ctx, interfaceName, toAdd); err != nil {
			return err
		}
	}

	return nil
}

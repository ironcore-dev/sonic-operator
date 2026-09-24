// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"fmt"
	"strings"
)

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

func (db *dbAccessor) EnsureInterfaceLoopback(ctx context.Context, loopbackName string) error {
	if err := db.configDB.HSet(ctx, "LOOPBACK_INTERFACE|"+loopbackName, "NULL", "NULL").Err(); err != nil {
		return fmt.Errorf("failed to ensure loopback interface %s: %w", loopbackName, err)
	}
	return nil
}

func (db *dbAccessor) DeleteInterfaceLoopback(ctx context.Context, loopbackName string) error {
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

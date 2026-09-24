// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"fmt"
	"strings"
)

func (db *dbAccessor) EnsureVLAN(ctx context.Context, vlanName string) error {
	vlanID := strings.TrimPrefix(vlanName, "Vlan")
	if err := db.configDB.HSet(ctx, "VLAN|"+vlanName, "vlanid", vlanID).Err(); err != nil {
		return fmt.Errorf("failed to ensure VLAN %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) EnsureVLANInterface(ctx context.Context, vlanName, vrfName string) error {
	key := "VLAN_INTERFACE|" + vlanName
	fields := []interface{}{"NULL", "NULL"}
	if vrfName != "" {
		fields = append(fields, "vrf_name", vrfName)
	}
	if err := db.configDB.HSet(ctx, key, fields...).Err(); err != nil {
		return fmt.Errorf("failed to ensure VLAN interface %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) DeleteVLAN(ctx context.Context, vlanName string) error {
	if err := db.configDB.Del(ctx, "VLAN|"+vlanName).Err(); err != nil {
		return fmt.Errorf("failed to delete VLAN %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) DeleteVLANInterface(ctx context.Context, vlanName string) error {
	if err := db.configDB.Del(ctx, "VLAN_INTERFACE|"+vlanName).Err(); err != nil {
		return fmt.Errorf("failed to delete VLAN interface %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) HasVLAN(ctx context.Context, vlanName string) (bool, error) {
	n, err := db.configDB.Exists(ctx, "VLAN|"+vlanName).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check VLAN %s: %w", vlanName, err)
	}
	return n > 0, nil
}

func (db *dbAccessor) SetVLANDescription(ctx context.Context, vlanName, description string) error {
	if err := db.configDB.HSet(ctx, "VLAN|"+vlanName, "description", description).Err(); err != nil {
		return fmt.Errorf("failed to set description for VLAN %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) EnsureVLANMember(ctx context.Context, vlanName, portName, taggingMode string) error {
	if err := db.configDB.HSet(ctx, "VLAN_MEMBER|"+vlanName+"|"+portName, "tagging_mode", taggingMode).Err(); err != nil {
		return fmt.Errorf("failed to ensure VLAN member %s on %s: %w", vlanName, portName, err)
	}
	return nil
}

func (db *dbAccessor) RemoveVLANMember(ctx context.Context, vlanName, portName string) error {
	if err := db.configDB.Del(ctx, "VLAN_MEMBER|"+vlanName+"|"+portName).Err(); err != nil {
		return fmt.Errorf("failed to remove VLAN member %s from %s: %w", vlanName, portName, err)
	}
	return nil
}

func (db *dbAccessor) ListVLANMembers(ctx context.Context, portName string) ([]string, error) {
	keys, err := scanKeys(ctx, db.configDB, "VLAN_MEMBER|*|"+portName)
	if err != nil {
		return nil, fmt.Errorf("failed to list VLAN members for %s: %w", portName, err)
	}
	vlans := make([]string, 0, len(keys))
	for _, k := range keys {
		parts := strings.SplitN(k, "|", 3)
		if len(parts) == 3 {
			vlans = append(vlans, parts[1])
		}
	}
	return vlans, nil
}

// SyncVLANMembers reconciles VLAN_MEMBER entries for portName toward desired state.
// desired maps vlanName to taggingMode; a nil map removes all memberships.
func (db *dbAccessor) SyncVLANMembers(ctx context.Context, portName string, desired map[string]string) error {
	current, err := db.ListVLANMembers(ctx, portName)
	if err != nil {
		return err
	}
	for _, vlan := range current {
		if _, ok := desired[vlan]; !ok {
			if err := db.RemoveVLANMember(ctx, vlan, portName); err != nil {
				return err
			}
		}
	}
	for vlan, taggingMode := range desired {
		if err := db.EnsureVLAN(ctx, vlan); err != nil {
			return err
		}
		if err := db.EnsureVLANMember(ctx, vlan, portName, taggingMode); err != nil {
			return err
		}
	}
	return nil
}

func (db *dbAccessor) EnsureDHCPRelay(ctx context.Context, vlanName string, servers []string) error {
	if err := db.configDB.HSet(ctx, "VLAN|"+vlanName, "dhcp_servers@", strings.Join(servers, ",")).Err(); err != nil {
		return fmt.Errorf("failed to set dhcp_servers for %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) DeleteDHCPRelay(ctx context.Context, vlanName string) error {
	if err := db.configDB.HDel(ctx, "VLAN|"+vlanName, "dhcp_servers@").Err(); err != nil {
		return fmt.Errorf("failed to delete dhcp_servers for %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) HasDHCPRelay(ctx context.Context, vlanName string) (bool, error) {
	result, err := db.configDB.HExists(ctx, "VLAN|"+vlanName, "dhcp_servers@").Result()
	if err != nil {
		return false, fmt.Errorf("failed to check dhcp_servers for %s: %w", vlanName, err)
	}
	return result, nil
}

// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
)

func (db *dbAccessor) EnsureVLAN(ctx context.Context, vlanName string) error {
	if err := validateVlanName(vlanName); err != nil {
		return err
	}
	vlanID := strings.TrimPrefix(vlanName, "Vlan")
	if err := db.configDB.HSet(ctx, "VLAN|"+vlanName, "vlanid", vlanID).Err(); err != nil {
		return fmt.Errorf("failed to ensure VLAN %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) EnsureVLANInterface(ctx context.Context, vlanName, vrfName string) error {
	if err := validateVlanName(vlanName); err != nil {
		return err
	}
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
	if err := validateVlanName(vlanName); err != nil {
		return err
	}
	if err := db.configDB.Del(ctx, "VLAN|"+vlanName).Err(); err != nil {
		return fmt.Errorf("failed to delete VLAN %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) DeleteVLANInterface(ctx context.Context, vlanName string) error {
	if err := validateVlanName(vlanName); err != nil {
		return err
	}
	if err := db.configDB.Del(ctx, "VLAN_INTERFACE|"+vlanName).Err(); err != nil {
		return fmt.Errorf("failed to delete VLAN interface %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) HasVLAN(ctx context.Context, vlanName string) (bool, error) {
	if err := validateVlanName(vlanName); err != nil {
		return false, err
	}
	n, err := db.configDB.Exists(ctx, "VLAN|"+vlanName).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check VLAN %s: %w", vlanName, err)
	}
	return n > 0, nil
}

func (db *dbAccessor) SetVLANDescription(ctx context.Context, vlanName, description string) error {
	if err := validateVlanName(vlanName); err != nil {
		return err
	}
	if err := db.configDB.HSet(ctx, "VLAN|"+vlanName, "description", description).Err(); err != nil {
		return fmt.Errorf("failed to set description for VLAN %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) EnsureVLANMember(ctx context.Context, vlanName, portName, taggingMode string) error {
	if err := validateVlanName(vlanName); err != nil {
		return err
	}
	if err := validatePortName(portName); err != nil {
		return err
	}
	if taggingMode != "tagged" && taggingMode != "untagged" {
		return fmt.Errorf("invalid tagging mode %q for %s on %s: must be tagged or untagged", taggingMode, vlanName, portName)
	}
	if err := db.configDB.HSet(ctx, "VLAN_MEMBER|"+vlanName+"|"+portName, "tagging_mode", taggingMode).Err(); err != nil {
		return fmt.Errorf("failed to ensure VLAN member %s on %s: %w", vlanName, portName, err)
	}
	return nil
}

func (db *dbAccessor) RemoveVLANMember(ctx context.Context, vlanName, portName string) error {
	if err := validateVlanName(vlanName); err != nil {
		return err
	}
	if err := validatePortName(portName); err != nil {
		return err
	}
	if err := db.configDB.Del(ctx, "VLAN_MEMBER|"+vlanName+"|"+portName).Err(); err != nil {
		return fmt.Errorf("failed to remove VLAN member %s from %s: %w", vlanName, portName, err)
	}
	return nil
}

func (db *dbAccessor) ListPortVLANs(ctx context.Context, portName string) ([]string, error) {
	if err := validatePortName(portName); err != nil {
		return nil, err
	}
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

// SyncPortVLANs reconciles VLAN_MEMBER entries for portName toward desired state.
// desired maps vlanName to taggingMode; a nil map removes all memberships.
// The caller is responsible for ensuring VLANs in desired already exist.
func (db *dbAccessor) SyncPortVLANs(ctx context.Context, portName string, desired map[string]string) error {
	if err := validatePortName(portName); err != nil {
		return err
	}
	for vlan, mode := range desired {
		if err := validateVlanName(vlan); err != nil {
			return err
		}
		if mode != "tagged" && mode != "untagged" {
			return fmt.Errorf("invalid tagging mode %q for %s on %s: must be tagged or untagged", mode, vlan, portName)
		}
	}
	current, err := db.ListPortVLANs(ctx, portName)
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
		if err := db.EnsureVLANMember(ctx, vlan, portName, taggingMode); err != nil {
			return err
		}
	}
	return nil
}

func (db *dbAccessor) EnsureDHCPRelay(ctx context.Context, vlanName string, servers []string) error {
	if err := validateVlanName(vlanName); err != nil {
		return err
	}
	if len(servers) == 0 {
		return fmt.Errorf("servers must not be empty")
	}
	for _, s := range servers {
		if _, err := netip.ParseAddr(s); err != nil {
			return fmt.Errorf("invalid DHCP server address %q: %w", s, err)
		}
	}
	if err := db.configDB.HSet(ctx, "VLAN|"+vlanName, "dhcp_servers@", strings.Join(servers, ",")).Err(); err != nil {
		return fmt.Errorf("failed to set dhcp_servers for %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) DeleteDHCPRelay(ctx context.Context, vlanName string) error {
	if err := validateVlanName(vlanName); err != nil {
		return err
	}
	if err := db.configDB.HDel(ctx, "VLAN|"+vlanName, "dhcp_servers@").Err(); err != nil {
		return fmt.Errorf("failed to delete dhcp_servers for %s: %w", vlanName, err)
	}
	return nil
}

func (db *dbAccessor) HasDHCPRelay(ctx context.Context, vlanName string) (bool, error) {
	if err := validateVlanName(vlanName); err != nil {
		return false, err
	}
	result, err := db.configDB.HExists(ctx, "VLAN|"+vlanName, "dhcp_servers@").Result()
	if err != nil {
		return false, fmt.Errorf("failed to check dhcp_servers for %s: %w", vlanName, err)
	}
	return result, nil
}

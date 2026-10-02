// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestAccessor(t *testing.T) (DBAccessor, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewDBAccessor(client, client, client), mr
}

// seedPort populates a CONFIG_DB PORT|<name> entry with realistic fields
// matching what a real SONiC switch returns (redis-cli -n 4 HGETALL 'PORT|Ethernet0').
func seedPort(mr *miniredis.Miniredis, name string, fields map[string]string) {
	defaults := map[string]string{
		"admin_status":    "up",
		"alias":           "fortyGigE0/0",
		"index":           "0",
		"lanes":           "25,26,27,28",
		"mtu":             "9100",
		"speed":           "40000",
		"dhcp_rate_limit": "300",
	}
	for k, v := range fields {
		defaults[k] = v
	}
	for k, v := range defaults {
		mr.HSet("PORT|"+name, k, v)
	}
}

func TestListPortNames(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedPort(mr, "Ethernet0", map[string]string{"index": "0"})
	seedPort(mr, "Ethernet4", map[string]string{"index": "1", "speed": "25000"})

	names, err := db.ListPortNames(context.Background())
	if err != nil {
		t.Fatalf("ListPortNames: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 ports, got %d: %v", len(names), names)
	}
	seen := map[string]bool{}
	for _, n := range names {
		seen[n] = true
	}
	if !seen["Ethernet0"] || !seen["Ethernet4"] {
		t.Errorf("unexpected port names: %v", names)
	}
}

func TestGetPortSpeed_Present(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedPort(mr, "Ethernet0", map[string]string{"speed": "40000"})

	speed, err := db.GetPortSpeed(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("GetPortSpeed: %v", err)
	}
	if speed != 40000 {
		t.Errorf("expected 40000, got %d", speed)
	}
}

func TestGetPortSpeed_Absent(t *testing.T) {
	db, _ := newTestAccessor(t)

	speed, err := db.GetPortSpeed(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("expected nil error for absent key, got: %v", err)
	}
	if speed != 0 {
		t.Errorf("expected 0, got %d", speed)
	}
}

func TestGetPortAlias_Present(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedPort(mr, "Ethernet0", map[string]string{"alias": "fortyGigE0/0"})

	alias, err := db.GetPortAlias(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("GetPortAlias: %v", err)
	}
	if alias != "fortyGigE0/0" {
		t.Errorf("expected fortyGigE0/0, got %q", alias)
	}
}

func TestHasPort_Present(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedPort(mr, "Ethernet0", nil)

	has, err := db.HasPort(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("HasPort: %v", err)
	}
	if !has {
		t.Error("expected HasPort to return true for seeded port")
	}
}

func TestHasPort_Absent(t *testing.T) {
	db, _ := newTestAccessor(t)

	has, err := db.HasPort(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("HasPort: %v", err)
	}
	if has {
		t.Error("expected HasPort to return false for absent port")
	}
}

// seedPortTable populates an APPL_DB PORT_TABLE:<name> entry with realistic fields
// matching what a real SONiC switch returns (redis-cli -n 0 HGETALL 'PORT_TABLE:Ethernet0').
func seedPortTable(mr *miniredis.Miniredis, name string, fields map[string]string) {
	defaults := map[string]string{
		"admin_status":    "up",
		"alias":           "fortyGigE0/0",
		"index":           "0",
		"lanes":           "25,26,27,28",
		"mtu":             "9100",
		"speed":           "40000",
		"dhcp_rate_limit": "300",
		"description":     "",
		"oper_status":     "up",
	}
	for k, v := range fields {
		defaults[k] = v
	}
	for k, v := range defaults {
		mr.HSet("PORT_TABLE:"+name, k, v)
	}
}

// seedStatePortTable populates a STATE_DB PORT_TABLE|<name> entry with realistic fields
// matching what a real SONiC switch returns (redis-cli -n 6 HGETALL 'PORT_TABLE|Ethernet0').
func seedStatePortTable(mr *miniredis.Miniredis, name string, fields map[string]string) {
	defaults := map[string]string{
		"state":              "ok",
		"netdev_oper_status": "up",
		"admin_status":       "up",
		"mtu":                "1500",
		"fec":                "none",
		"supported_speeds":   "",
		"host_tx_ready":      "true",
		"effective_admin":    "up",
		"speed":              "100000",
	}
	for k, v := range fields {
		defaults[k] = v
	}
	for k, v := range defaults {
		mr.HSet("PORT_TABLE|"+name, k, v)
	}
}

func TestGetAdminStatus_EthernetUp(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedPortTable(mr, "Ethernet0", map[string]string{"admin_status": "up"})

	up, err := db.GetAdminStatus(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("GetAdminStatus: %v", err)
	}
	if !up {
		t.Error("expected admin_status up")
	}
}

func TestGetAdminStatus_EthernetDown(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedPortTable(mr, "Ethernet0", map[string]string{"admin_status": "down"})

	up, err := db.GetAdminStatus(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("GetAdminStatus: %v", err)
	}
	if up {
		t.Error("expected admin_status down")
	}
}

func TestGetOperStatus_EthernetUp(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedPortTable(mr, "Ethernet0", map[string]string{"oper_status": "up"})

	up, err := db.GetOperStatus(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("GetOperStatus: %v", err)
	}
	if !up {
		t.Error("expected oper_status up")
	}
}

func TestGetOperStatus_EthernetDown(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedPortTable(mr, "Ethernet0", map[string]string{"oper_status": "down"})

	up, err := db.GetOperStatus(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("GetOperStatus: %v", err)
	}
	if up {
		t.Error("expected oper_status down")
	}
}

func TestListPortTableEntries(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedPortTable(mr, "Ethernet0", map[string]string{"speed": "40000", "oper_status": "up"})
	seedPortTable(mr, "Ethernet4", map[string]string{"speed": "25000", "oper_status": "down"})

	entries, err := db.ListPortTableEntries(context.Background())
	if err != nil {
		t.Fatalf("ListPortTableEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries["Ethernet0"]["speed"] != "40000" {
		t.Errorf("Ethernet0 speed: got %q, want 40000", entries["Ethernet0"]["speed"])
	}
	if entries["Ethernet0"]["oper_status"] != "up" {
		t.Errorf("Ethernet0 oper_status: got %q, want up", entries["Ethernet0"]["oper_status"])
	}
	if entries["Ethernet4"]["oper_status"] != "down" {
		t.Errorf("Ethernet4 oper_status: got %q, want down", entries["Ethernet4"]["oper_status"])
	}
}

func TestGetLLDPEntry_Present(t *testing.T) {
	db, mr := newTestAccessor(t)
	mr.HSet("LLDP_ENTRY_TABLE:Ethernet48",
		"lldp_rem_port_id_subtype", "5",
		"lldp_rem_port_id", "ens2f0np0",
		"lldp_rem_port_desc", "",
		"lldp_rem_chassis_id_subtype", "7",
		"lldp_rem_chassis_id", "4c4c4544003058108054c8c04f423033",
		"lldp_rem_sys_name", "fra4-lab-dell-5",
		"lldp_rem_sys_desc", "",
		"lldp_rem_man_addr", "",
		"lldp_rem_time_mark", "48",
		"lldp_rem_index", "403",
		"lldp_rem_sys_cap_supported", "28 00",
		"lldp_rem_sys_cap_enabled", "00 00",
	)

	entry, err := db.GetLLDPEntry(context.Background(), "Ethernet48")
	if err != nil {
		t.Fatalf("GetLLDPEntry: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry["lldp_rem_sys_name"] != "fra4-lab-dell-5" {
		t.Errorf("lldp_rem_sys_name: got %q, want fra4-lab-dell-5", entry["lldp_rem_sys_name"])
	}
	if entry["lldp_rem_port_id"] != "ens2f0np0" {
		t.Errorf("lldp_rem_port_id: got %q, want ens2f0np0", entry["lldp_rem_port_id"])
	}
	if entry["lldp_rem_chassis_id"] != "4c4c4544003058108054c8c04f423033" {
		t.Errorf("lldp_rem_chassis_id: got %q, want 4c4c4544003058108054c8c04f423033", entry["lldp_rem_chassis_id"])
	}
}

func TestGetLLDPEntry_Absent(t *testing.T) {
	db, _ := newTestAccessor(t)

	entry, err := db.GetLLDPEntry(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("expected nil error for absent entry, got: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil map for absent LLDP entry, got: %v", entry)
	}
}

func TestGetPortSupportedSpeeds_Present(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedStatePortTable(mr, "Ethernet0", map[string]string{"supported_speeds": "10000,25000,100000"})

	speeds, err := db.GetPortSupportedSpeeds(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("GetPortSupportedSpeeds: %v", err)
	}
	if speeds != "10000,25000,100000" {
		t.Errorf("unexpected supported speeds: %q", speeds)
	}
}

func TestGetPortSupportedSpeeds_Absent(t *testing.T) {
	db, _ := newTestAccessor(t)

	speeds, err := db.GetPortSupportedSpeeds(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("expected nil error for absent key, got: %v", err)
	}
	if speeds != "" {
		t.Errorf("expected empty string, got: %q", speeds)
	}
}

func TestGetPortSupportedSpeeds_EmptyOnVirtual(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedStatePortTable(mr, "Ethernet0", nil) // supported_speeds is "" on virtual/copper ports

	speeds, err := db.GetPortSupportedSpeeds(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("GetPortSupportedSpeeds: %v", err)
	}
	if speeds != "" {
		t.Errorf("expected empty string for virtual port, got: %q", speeds)
	}
}

// seedTransceiverInfo populates a STATE_DB TRANSCEIVER_INFO|<name> entry with realistic fields
// matching what a real SONiC switch returns (redis-cli -n 6 HGETALL 'TRANSCEIVER_INFO|Ethernet0').
func seedTransceiverInfo(mr *miniredis.Miniredis, name string, fields map[string]string) {
	defaults := map[string]string{
		"type":             "QSFP28 or later",
		"manufacturer":     "Switch2Open     ",
		"model":            "S-D-Q-100G-2    ",
		"serial":           "GE2509270240    ",
		"vendor_rev":       "X1",
		"vendor_oui":       "00-00-00",
		"connector":        "No separable connector",
		"encoding":         "Unspecified",
		"nominal_bit_rate": "255",
		"cable_type":       "Length Cable Assembly(m)",
		"cable_length":     "2.0",
		"vendor_date":      "2025-10-09   ",
		"is_replaceable":   "True",
	}
	for k, v := range fields {
		defaults[k] = v
	}
	for k, v := range defaults {
		mr.HSet("TRANSCEIVER_INFO|"+name, k, v)
	}
}

func TestGetTransceiverType_Present(t *testing.T) {
	db, mr := newTestAccessor(t)
	seedTransceiverInfo(mr, "Ethernet0", nil)

	typ, err := db.GetTransceiverType(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("GetTransceiverType: %v", err)
	}
	if typ != "QSFP28 or later" {
		t.Errorf("unexpected type: %q", typ)
	}
}

func TestGetTransceiverType_Absent(t *testing.T) {
	db, _ := newTestAccessor(t)

	typ, err := db.GetTransceiverType(context.Background(), "Ethernet0")
	if err != nil {
		t.Fatalf("expected nil error for absent key, got: %v", err)
	}
	if typ != "" {
		t.Errorf("expected empty string, got: %q", typ)
	}
}

func TestSetFEC_Valid(t *testing.T) {
	db, _ := newTestAccessor(t)
	for _, fec := range []string{"rs", "fc", "none"} {
		if err := db.SetFEC(context.Background(), "Ethernet0", fec); err != nil {
			t.Errorf("SetFEC(%q): unexpected error: %v", fec, err)
		}
	}
}

func TestSetFEC_Invalid(t *testing.T) {
	db, _ := newTestAccessor(t)
	if err := db.SetFEC(context.Background(), "Ethernet0", "auto"); err == nil {
		t.Error("SetFEC with invalid value should return error")
	}
}

func TestSetAdminStatus_Valid(t *testing.T) {
	db, _ := newTestAccessor(t)
	for _, status := range []string{"up", "down"} {
		if err := db.SetAdminStatus(context.Background(), "Ethernet0", status); err != nil {
			t.Errorf("SetAdminStatus(%q): unexpected error: %v", status, err)
		}
	}
}

func TestSetAdminStatus_Invalid(t *testing.T) {
	db, _ := newTestAccessor(t)
	if err := db.SetAdminStatus(context.Background(), "Ethernet0", "enabled"); err == nil {
		t.Error("SetAdminStatus with invalid value should return error")
	}
	if err := db.SetAdminStatus(context.Background(), "Ethernet0", ""); err == nil {
		t.Error("SetAdminStatus with empty value should return error")
	}
	if err := db.SetAdminStatus(context.Background(), "Unknown0", "up"); err == nil {
		t.Error("SetAdminStatus with unknown interface prefix should return error")
	}
}

func TestAddIPAddresses_InvalidPrefix(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	for _, bad := range []string{"not-an-ip", "10.0.0.1", "2001:db8::1"} {
		if err := db.AddIPAddresses(ctx, "Loopback0", []string{bad}); err == nil {
			t.Errorf("AddIPAddresses(%q): expected error for invalid prefix", bad)
		}
	}
}

func TestAddIPAddresses_ValidPrefix(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	for _, good := range []string{"10.0.0.1/32", "192.168.1.0/24", "2001:db8::1/128", "fd00::/48"} {
		if err := db.AddIPAddresses(ctx, "Loopback0", []string{good}); err != nil {
			t.Errorf("AddIPAddresses(%q): unexpected error: %v", good, err)
		}
	}
}

func TestSyncIPAddresses(t *testing.T) {
	db, mr := newTestAccessor(t)
	ctx := context.Background()
	iface := "Loopback0"

	// seed two existing IPs
	mr.HSet("LOOPBACK_INTERFACE|"+iface+"|10.0.0.1/32", "NULL", "NULL")
	mr.HSet("LOOPBACK_INTERFACE|"+iface+"|10.0.0.2/32", "NULL", "NULL")

	// sync to keep only 10.0.0.2 and add 10.0.0.3
	desired := []string{"10.0.0.2/32", "10.0.0.3/32"}
	if err := db.SyncIPAddresses(ctx, iface, desired); err != nil {
		t.Fatalf("SyncIPAddresses: %v", err)
	}

	addrs, err := db.ListIPAddresses(ctx, iface)
	if err != nil {
		t.Fatalf("ListIPAddresses: %v", err)
	}
	addrSet := map[string]bool{}
	for _, a := range addrs {
		addrSet[a] = true
	}
	if addrSet["10.0.0.1/32"] {
		t.Error("10.0.0.1/32 should have been removed")
	}
	if !addrSet["10.0.0.2/32"] {
		t.Error("10.0.0.2/32 should be present")
	}
	if !addrSet["10.0.0.3/32"] {
		t.Error("10.0.0.3/32 should have been added")
	}
}

func TestSyncPortVLANs(t *testing.T) {
	db, mr := newTestAccessor(t)
	ctx := context.Background()
	port := "Ethernet0"

	// seed existing memberships
	mr.HSet("VLAN_MEMBER|Vlan10|"+port, "tagging_mode", "untagged")
	mr.HSet("VLAN_MEMBER|Vlan20|"+port, "tagging_mode", "tagged")
	mr.HSet("VLAN|Vlan10", "vlanid", "10")
	mr.HSet("VLAN|Vlan20", "vlanid", "20")
	mr.HSet("VLAN|Vlan30", "vlanid", "30")

	// sync: remove Vlan10, keep Vlan20, add Vlan30
	desired := map[string]string{
		"Vlan20": "tagged",
		"Vlan30": "untagged",
	}
	if err := db.SyncPortVLANs(ctx, port, desired); err != nil {
		t.Fatalf("SyncPortVLANs: %v", err)
	}

	members, err := db.ListPortVLANs(ctx, port)
	if err != nil {
		t.Fatalf("ListPortVLANs: %v", err)
	}
	memberSet := map[string]bool{}
	for _, m := range members {
		memberSet[m] = true
	}
	if memberSet["Vlan10"] {
		t.Error("Vlan10 membership should have been removed")
	}
	if !memberSet["Vlan20"] {
		t.Error("Vlan20 membership should be present")
	}
	if !memberSet["Vlan30"] {
		t.Error("Vlan30 membership should have been added")
	}
}

func TestRemoveIPAddresses_InvalidPrefix(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	for _, bad := range []string{"not-an-ip", "10.0.0.1", "2001:db8::1"} {
		if err := db.RemoveIPAddresses(ctx, "Loopback0", []string{bad}); err == nil {
			t.Errorf("RemoveIPAddresses(%q): expected error for invalid prefix", bad)
		}
	}
}

func TestEnsureLoopback_InvalidName(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	for _, bad := range []string{"Ethernet0", "Vlan10", "", "lo0"} {
		if err := db.EnsureLoopback(ctx, bad); err == nil {
			t.Errorf("EnsureLoopback(%q): expected error for invalid name", bad)
		}
	}
}

func TestDeleteLoopback_InvalidName(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	if err := db.DeleteLoopback(ctx, "Ethernet0"); err == nil {
		t.Error("DeleteLoopback with non-loopback name should return error")
	}
}

func TestSetMTU_Validation(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	if err := db.SetMTU(ctx, "Ethernet0", 0); err == nil {
		t.Error("SetMTU with zero MTU should return error")
	}
	if err := db.SetMTU(ctx, "Ethernet0", 99999); err == nil {
		t.Error("SetMTU with oversized MTU should return error")
	}
	if err := db.SetMTU(ctx, "Loopback0", 1500); err == nil {
		t.Error("SetMTU on loopback (invalid port name) should return error")
	}
	if err := db.SetMTU(ctx, "Ethernet0", 1500); err != nil {
		t.Errorf("SetMTU with valid args: unexpected error: %v", err)
	}
}

func TestSetPortAlias_Validation(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	if err := db.SetPortAlias(ctx, "Ethernet0", ""); err == nil {
		t.Error("SetPortAlias with empty alias should return error")
	}
	if err := db.SetPortAlias(ctx, "BadPort0", "alias"); err == nil {
		t.Error("SetPortAlias with invalid port name should return error")
	}
	if err := db.SetPortAlias(ctx, "Ethernet0", "etp0"); err != nil {
		t.Errorf("SetPortAlias with valid args: unexpected error: %v", err)
	}
}

func TestSetHostname_Empty(t *testing.T) {
	db, _ := newTestAccessor(t)
	if err := db.SetHostname(context.Background(), ""); err == nil {
		t.Error("SetHostname with empty string should return error")
	}
}

func TestEnsureVLAN_InvalidName(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	for _, bad := range []string{"vlan10", "VLAN10", "Ethernet0", "Vlan0", "Vlan4095", "Vlanfoo", ""} {
		if err := db.EnsureVLAN(ctx, bad); err == nil {
			t.Errorf("EnsureVLAN(%q): expected error for invalid name", bad)
		}
	}
	for _, good := range []string{"Vlan1", "Vlan100", "Vlan4094"} {
		if err := db.EnsureVLAN(ctx, good); err != nil {
			t.Errorf("EnsureVLAN(%q): unexpected error: %v", good, err)
		}
	}
}

func TestEnsureVLANMember_Validation(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	if err := db.EnsureVLANMember(ctx, "BadVlan", "Ethernet0", "tagged"); err == nil {
		t.Error("EnsureVLANMember with invalid VLAN name should return error")
	}
	if err := db.EnsureVLANMember(ctx, "Vlan10", "lo0", "tagged"); err == nil {
		t.Error("EnsureVLANMember with invalid port name should return error")
	}
	if err := db.EnsureVLANMember(ctx, "Vlan10", "Ethernet0", "access"); err == nil {
		t.Error("EnsureVLANMember with invalid tagging mode should return error")
	}
	if err := db.EnsureVLANMember(ctx, "Vlan10", "Ethernet0", "tagged"); err != nil {
		t.Errorf("EnsureVLANMember with valid args: unexpected error: %v", err)
	}
}

func TestEnsureDHCPRelay_Validation(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	if err := db.EnsureDHCPRelay(ctx, "BadVlan", []string{"192.168.1.1"}); err == nil {
		t.Error("EnsureDHCPRelay with invalid VLAN name should return error")
	}
	if err := db.EnsureDHCPRelay(ctx, "Vlan10", []string{}); err == nil {
		t.Error("EnsureDHCPRelay with empty servers should return error")
	}
	if err := db.EnsureDHCPRelay(ctx, "Vlan10", []string{"not-an-ip"}); err == nil {
		t.Error("EnsureDHCPRelay with invalid server IP should return error")
	}
	if err := db.EnsureDHCPRelay(ctx, "Vlan10", []string{"192.168.1.1", "2001:db8::1"}); err != nil {
		t.Errorf("EnsureDHCPRelay with valid args: unexpected error: %v", err)
	}
}

func TestSyncPortVLANs_Validation(t *testing.T) {
	db, _ := newTestAccessor(t)
	ctx := context.Background()
	if err := db.SyncPortVLANs(ctx, "lo0", map[string]string{"Vlan10": "tagged"}); err == nil {
		t.Error("SyncPortVLANs with invalid port name should return error")
	}
	if err := db.SyncPortVLANs(ctx, "Ethernet0", map[string]string{"BadVlan": "tagged"}); err == nil {
		t.Error("SyncPortVLANs with invalid VLAN name should return error")
	}
	if err := db.SyncPortVLANs(ctx, "Ethernet0", map[string]string{"Vlan10": "access"}); err == nil {
		t.Error("SyncPortVLANs with invalid tagging mode should return error")
	}
}

func TestParseSupportedSpeeds(t *testing.T) {
	cases := []struct {
		raw      string
		fallback int
		want     []int32
	}{
		{"10000,25000,100000", 0, []int32{10, 25, 100}},
		{"", 10000, []int32{10}},
		{"", 0, nil},
		{"  25000 , 100000 ", 0, []int32{25, 100}},
	}
	for _, c := range cases {
		got := ParseSupportedSpeeds(c.raw, c.fallback)
		if len(got) != len(c.want) {
			t.Errorf("ParseSupportedSpeeds(%q, %d): got %v, want %v", c.raw, c.fallback, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("ParseSupportedSpeeds(%q, %d)[%d]: got %d, want %d", c.raw, c.fallback, i, got[i], c.want[i])
			}
		}
	}
}

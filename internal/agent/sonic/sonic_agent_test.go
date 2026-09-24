// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"testing"
)

func TestVlanPrefixFromLoopback(t *testing.T) {
	tests := []struct {
		name         string
		loopbackCIDR string
		vlanID       int32
		want         string
		wantErr      bool
	}{
		{
			name:         "vlan 1001",
			loopbackCIDR: "2001:db8:0:1::1/128",
			vlanID:       1001,
			want:         "2001:db8:0:3e9::/80",
		},
		{
			name:         "vlan 1",
			loopbackCIDR: "2001:db8:0:1::1/128",
			vlanID:       1,
			want:         "2001:db8:0:1::/80",
		},
		{
			name:         "vlan 0",
			loopbackCIDR: "2001:db8:0:1::1/128",
			vlanID:       0,
			want:         "2001:db8::/80",
		},
		{
			name:         "different /48 base",
			loopbackCIDR: "fd00:ab:cd::1/128",
			vlanID:       2,
			want:         "fd00:ab:cd:2::/80",
		},
		{
			name:         "invalid CIDR",
			loopbackCIDR: "notanip",
			vlanID:       1,
			wantErr:      true,
		},
		{
			name:         "IPv4 address",
			loopbackCIDR: "192.168.1.1/32",
			vlanID:       1,
			wantErr:      true,
		},
		{
			name:         "negative vlan ID",
			loopbackCIDR: "2001:db8::1/128",
			vlanID:       -1,
			wantErr:      true,
		},
		{
			name:         "vlan ID overflow",
			loopbackCIDR: "2001:db8::1/128",
			vlanID:       0x10000,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vlanPrefixFromLoopback(tt.loopbackCIDR, tt.vlanID)
			if (err != nil) != tt.wantErr {
				t.Errorf("vlanPrefixFromLoopback(%q, %d) error = %v, wantErr %v", tt.loopbackCIDR, tt.vlanID, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("vlanPrefixFromLoopback(%q, %d) = %q, want %q", tt.loopbackCIDR, tt.vlanID, got, tt.want)
			}
		})
	}
}

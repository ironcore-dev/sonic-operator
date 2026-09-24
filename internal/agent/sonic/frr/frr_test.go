// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package frr_test

import (
	"strings"
	"testing"

	"github.com/ironcore-dev/sonic-operator/internal/agent/sonic/frr"
	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestFRR(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "FRR Suite")
}

func leafBGPConfig() *agent.FabricBGPConfig {
	return &agent.FabricBGPConfig{
		ASN: 100,
		PeerGroups: []agent.FabricBGPPeerGroup{
			{
				Name: "NORTH",
				Neighbors: []agent.FabricBGPNeighbor{
					{InterfaceID: "Ethernet120"},
					{InterfaceID: "Ethernet124"},
				},
			},
			{
				Name: "SOUTH",
				Neighbors: []agent.FabricBGPNeighbor{
					{InterfaceID: "Vlan1001", VlanID: 1001},
					{InterfaceID: "Vlan1002", VlanID: 1002},
				},
			},
		},
	}
}

var _ = ginkgo.Describe("GenerateFRRConfig", func() {

	// ── error cases ──────────────────────────────────────────────────────────

	ginkgo.It("returns error when BGP config is nil", func() {
		_, err := frr.GenerateFRRConfig(nil, "leaf-1", nil)
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	// ── header / global directives ───────────────────────────────────────────

	ginkgo.It("produces valid frr header lines", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("frr version 8.1"))
		gomega.Expect(out).To(gomega.ContainSubstring("frr defaults traditional"))
		gomega.Expect(out).To(gomega.ContainSubstring("service integrated-vtysh-config"))
	})

	ginkgo.It("sets hostname", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "my-leaf", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("hostname my-leaf"))
	})

	// ── router bgp block ─────────────────────────────────────────────────────

	ginkgo.It("includes the correct ASN in router bgp statement", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("router bgp 100"))
	})

	ginkgo.It("derives router-id from ASN as 1.0.(ASN/256).(ASN%256) for small ASN", func() {
		// ASN=100 → 1.0.0.100
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("bgp router-id 1.0.0.100"))
	})

	ginkgo.It("derives router-id correctly for ASN > 255", func() {
		// 300/256=1, 300%256=44 → 1.0.1.44
		cfg := &agent.FabricBGPConfig{ASN: 300, PeerGroups: []agent.FabricBGPPeerGroup{{Name: "NORTH"}}}
		out, err := frr.GenerateFRRConfig(cfg, "leaf-2", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("bgp router-id 1.0.1.44"))
	})

	ginkgo.It("derives router-id correctly for ASN=256", func() {
		// 256/256=1, 256%256=0 → 1.0.1.0
		cfg := &agent.FabricBGPConfig{ASN: 256, PeerGroups: []agent.FabricBGPPeerGroup{{Name: "NORTH"}}}
		out, err := frr.GenerateFRRConfig(cfg, "leaf-3", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("bgp router-id 1.0.1.0"))
	})

	ginkgo.It("includes required bgp global flags", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("no bgp ebgp-requires-policy"))
		gomega.Expect(out).To(gomega.ContainSubstring("no bgp default ipv4-unicast"))
		gomega.Expect(out).To(gomega.ContainSubstring("bgp bestpath as-path multipath-relax"))
		gomega.Expect(out).To(gomega.ContainSubstring("no bgp network import-check"))
	})

	// ── peer groups ──────────────────────────────────────────────────────────

	ginkgo.It("defines NORTH peer-group with timers", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor NORTH peer-group"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor NORTH remote-as external"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor NORTH timers 3 9"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor NORTH timers connect 20"))
	})

	ginkgo.It("defines SOUTH peer-group with timers", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor SOUTH peer-group"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor SOUTH remote-as external"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor SOUTH timers 3 9"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor SOUTH timers connect 20"))
	})

	ginkgo.It("assigns NORTH neighbors to NORTH peer-group", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor Ethernet120 interface peer-group NORTH"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor Ethernet124 interface peer-group NORTH"))
	})

	ginkgo.It("assigns SOUTH neighbors to SOUTH peer-group", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor Vlan1001 interface peer-group SOUTH"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor Vlan1002 interface peer-group SOUTH"))
	})

	ginkgo.It("is case-insensitive for peer group name matching", func() {
		cfg := &agent.FabricBGPConfig{
			ASN: 100,
			PeerGroups: []agent.FabricBGPPeerGroup{
				{Name: "north", Neighbors: []agent.FabricBGPNeighbor{{InterfaceID: "Ethernet0"}}},
			},
		}
		out, err := frr.GenerateFRRConfig(cfg, "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor Ethernet0 interface peer-group NORTH"))
	})

	ginkgo.It("produces no NORTH neighbor lines when NORTH peer group is absent", func() {
		cfg := &agent.FabricBGPConfig{
			ASN: 100,
			PeerGroups: []agent.FabricBGPPeerGroup{
				{Name: "SOUTH", Neighbors: []agent.FabricBGPNeighbor{{InterfaceID: "Vlan1", VlanID: 1}}},
			},
		}
		out, err := frr.GenerateFRRConfig(cfg, "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).NotTo(gomega.ContainSubstring("peer-group NORTH\n"))
		// SOUTH must still appear
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor Vlan1 interface peer-group SOUTH"))
	})

	ginkgo.It("produces no neighbor lines when peer groups are empty", func() {
		cfg := &agent.FabricBGPConfig{ASN: 42}
		out, err := frr.GenerateFRRConfig(cfg, "spine-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).NotTo(gomega.ContainSubstring("interface peer-group"))
	})

	// ── VLAN interface stanzas ───────────────────────────────────────────────

	ginkgo.It("emits interface stanzas with RA suppression for each VLAN neighbor", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("interface Vlan1001"))
		gomega.Expect(out).To(gomega.ContainSubstring("interface Vlan1002"))
		gomega.Expect(strings.Count(out, "no ipv6 nd suppress-ra")).To(gomega.Equal(2))
		gomega.Expect(strings.Count(out, "ipv6 nd managed-config-flag")).To(gomega.Equal(2))
		gomega.Expect(strings.Count(out, "ipv6 nd other-config-flag")).To(gomega.Equal(2))
	})

	ginkgo.It("does not emit VLAN interface stanzas when no VLAN neighbors exist", func() {
		cfg := &agent.FabricBGPConfig{
			ASN: 50,
			PeerGroups: []agent.FabricBGPPeerGroup{
				{Name: "NORTH", Neighbors: []agent.FabricBGPNeighbor{{InterfaceID: "Ethernet0"}}},
			},
		}
		out, err := frr.GenerateFRRConfig(cfg, "spine-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).NotTo(gomega.ContainSubstring("no ipv6 nd suppress-ra"))
	})

	ginkgo.It("deduplicates VLAN IDs across peer groups", func() {
		cfg := &agent.FabricBGPConfig{
			ASN: 100,
			PeerGroups: []agent.FabricBGPPeerGroup{
				{Name: "SOUTH", Neighbors: []agent.FabricBGPNeighbor{
					{InterfaceID: "Vlan5", VlanID: 5},
					{InterfaceID: "Vlan5b", VlanID: 5}, // duplicate VlanID
				}},
			},
		}
		out, err := frr.GenerateFRRConfig(cfg, "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(strings.Count(out, "interface Vlan5")).To(gomega.Equal(1))
	})

	// ── address-family and prefixes ──────────────────────────────────────────

	ginkgo.It("includes advertised prefixes as network statements", func() {
		prefixes := []string{"fd00::/48", "fd00:1::/48"}
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", prefixes)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("network fd00::/48"))
		gomega.Expect(out).To(gomega.ContainSubstring("network fd00:1::/48"))
	})

	ginkgo.It("includes advertised prefixes as static reject routes", func() {
		prefixes := []string{"fd00::/48", "fd00:1::/48"}
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", prefixes)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("ipv6 route fd00::/48 reject"))
		gomega.Expect(out).To(gomega.ContainSubstring("ipv6 route fd00:1::/48 reject"))
	})

	ginkgo.It("activates both NORTH and SOUTH in address-family ipv6 unicast", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("address-family ipv6 unicast"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor NORTH activate"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor SOUTH activate"))
		gomega.Expect(out).To(gomega.ContainSubstring("exit-address-family"))
	})

	ginkgo.It("applies route-maps to NORTH in address-family", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor NORTH route-map RM_NORTH_IN in"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor NORTH route-map RM_NORTH_OUT out"))
	})

	ginkgo.It("applies route-maps to SOUTH in address-family", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor SOUTH route-map RM_SOUTH_IN in"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor SOUTH route-map RM_SOUTH_OUT out"))
	})

	// ── route-maps ───────────────────────────────────────────────────────────

	ginkgo.It("includes all route-map definitions", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("route-map RM_NORTH_IN permit 10"))
		gomega.Expect(out).To(gomega.ContainSubstring("route-map RM_NORTH_OUT deny 10"))
		gomega.Expect(out).To(gomega.ContainSubstring("route-map RM_NORTH_OUT permit 20"))
		gomega.Expect(out).To(gomega.ContainSubstring("route-map RM_SOUTH_IN permit 10"))
		gomega.Expect(out).To(gomega.ContainSubstring("route-map RM_SOUTH_OUT permit 10"))
	})

	ginkgo.It("includes bgp community-list for NORTH filtering", func() {
		out, err := frr.GenerateFRRConfig(leafBGPConfig(), "leaf-1", nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring("set community 65000:100"))
		gomega.Expect(out).To(gomega.ContainSubstring("bgp community-list 10 permit 65000:100"))
		gomega.Expect(out).To(gomega.ContainSubstring("match community 10"))
	})

	// ── golden output ────────────────────────────────────────────────────────

	ginkgo.It("renders the full expected output for a minimal spine config", func() {
		cfg := &agent.FabricBGPConfig{
			ASN: 200,
			PeerGroups: []agent.FabricBGPPeerGroup{
				{Name: "NORTH", Neighbors: []agent.FabricBGPNeighbor{
					{InterfaceID: "Ethernet0"},
				}},
			},
		}
		out, err := frr.GenerateFRRConfig(cfg, "spine-1", []string{"fd00::/48"})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Verify structural ordering: header before router bgp before address-family before route-maps
		headerPos := strings.Index(out, "frr version 8.1")
		routerPos := strings.Index(out, "router bgp 200")
		afPos := strings.Index(out, "address-family ipv6 unicast")
		rmPos := strings.Index(out, "route-map RM_NORTH_IN")
		gomega.Expect(headerPos).To(gomega.BeNumerically("<", routerPos))
		gomega.Expect(routerPos).To(gomega.BeNumerically("<", afPos))
		gomega.Expect(afPos).To(gomega.BeNumerically("<", rmPos))

		// 200/256=0, 200%256=200 → 1.0.0.200
		gomega.Expect(out).To(gomega.ContainSubstring("bgp router-id 1.0.0.200"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor Ethernet0 interface peer-group NORTH"))
		gomega.Expect(out).To(gomega.ContainSubstring("network fd00::/48"))
		gomega.Expect(out).To(gomega.ContainSubstring("ipv6 route fd00::/48 reject"))
		// No VLAN stanzas for a spine
		gomega.Expect(out).NotTo(gomega.ContainSubstring("no ipv6 nd suppress-ra"))
	})

	ginkgo.It("renders correct output for a spine config with LEAFS peer group", func() {
		cfg := &agent.FabricBGPConfig{
			ASN: 200,
			PeerGroups: []agent.FabricBGPPeerGroup{
				{Name: "LEAFS", Neighbors: []agent.FabricBGPNeighbor{
					{InterfaceID: "Ethernet0"},
					{InterfaceID: "Ethernet4"},
				}},
			},
		}
		out, err := frr.GenerateFRRConfig(cfg, "spine-1", []string{"fd00::/48"})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// 200/256=0, 200%256=200 → spine uses 2.0 prefix
		gomega.Expect(out).To(gomega.ContainSubstring("bgp router-id 2.0.0.200"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor LEAFS peer-group"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor Ethernet0 interface peer-group LEAFS"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor Ethernet4 interface peer-group LEAFS"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor LEAFS activate"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor LEAFS route-map RM_LEAFS_IN in"))
		gomega.Expect(out).To(gomega.ContainSubstring("neighbor LEAFS route-map RM_LEAFS_OUT out"))
		// Pass-through route-maps — no community filtering
		gomega.Expect(out).To(gomega.ContainSubstring("route-map RM_LEAFS_IN permit 10"))
		gomega.Expect(out).To(gomega.ContainSubstring("route-map RM_LEAFS_OUT permit 10"))
		gomega.Expect(out).NotTo(gomega.ContainSubstring("set community"))
		gomega.Expect(out).NotTo(gomega.ContainSubstring("deny"))
		// No NORTH/SOUTH
		gomega.Expect(out).NotTo(gomega.ContainSubstring("neighbor NORTH"))
		gomega.Expect(out).NotTo(gomega.ContainSubstring("neighbor SOUTH"))
		// No VLAN stanzas
		gomega.Expect(out).NotTo(gomega.ContainSubstring("no ipv6 nd suppress-ra"))
	})
})

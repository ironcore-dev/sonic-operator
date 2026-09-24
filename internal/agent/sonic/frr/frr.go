// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package frr

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"
)

//go:embed frr.conf.tmpl
var frrTemplate string

var tmpl = template.Must(template.New("frr.conf").Parse(frrTemplate))

type peerGroupData struct {
	Name        string
	Neighbors   []string
	CommunityIn string // non-empty → emit community set + deny/permit route-maps
}

type frrConfigData struct {
	Hostname   string
	ASN        uint32
	RouterID   string
	VLANIDs    []int32
	PeerGroups []peerGroupData
	Prefixes   []string
}

// GenerateFRRConfig renders the frr.conf template from the given BGP config,
// hostname, and advertised prefixes.
//
// Topology is inferred from peer group names:
//   - Any group named NORTH (case-insensitive) → leaf: router-id 1.0.x.x, NORTH gets community filtering
//   - No NORTH group → spine: router-id 2.0.x.x, all groups are pass-through
//
// VLAN interface stanzas are emitted for every unique non-zero VlanID.
func GenerateFRRConfig(cfg *agent.FabricBGPConfig, hostname string, prefixes []string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("BGP config is nil")
	}

	// Determine topology from peer group names.
	isLeaf := false
	for _, pg := range cfg.PeerGroups {
		if strings.ToUpper(pg.Name) == "NORTH" {
			isLeaf = true
			break
		}
	}

	routerIDPrefix := "2.0"
	if isLeaf {
		routerIDPrefix = "1.0"
	}

	routerID := cfg.RouterID
	if routerID == "" {
		routerID = fmt.Sprintf("%s.%d.%d", routerIDPrefix, cfg.ASN/256, cfg.ASN%256)
	}

	data := frrConfigData{
		Hostname: hostname,
		ASN:      cfg.ASN,
		RouterID: routerID,
		Prefixes: prefixes,
	}

	vlanSet := make(map[int32]struct{})
	for _, pg := range cfg.PeerGroups {
		upper := strings.ToUpper(pg.Name)
		neighbors := make([]string, 0, len(pg.Neighbors))
		for _, n := range pg.Neighbors {
			if n.InterfaceID != "" {
				neighbors = append(neighbors, n.InterfaceID)
			}
			if n.VlanID != 0 {
				vlanSet[n.VlanID] = struct{}{}
			}
		}
		pgd := peerGroupData{
			Name:      upper,
			Neighbors: neighbors,
		}
		if upper == "NORTH" {
			pgd.CommunityIn = "65000:100"
		}
		data.PeerGroups = append(data.PeerGroups, pgd)
	}

	for id := range vlanSet {
		data.VLANIDs = append(data.VLANIDs, id)
	}
	sort.Slice(data.VLANIDs, func(i, j int) bool { return data.VLANIDs[i] < data.VLANIDs[j] })

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("rendering frr.conf template: %w", err)
	}
	return sb.String(), nil
}

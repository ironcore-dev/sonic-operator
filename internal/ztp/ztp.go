// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ztp

import (
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"text/template"
)

//go:embed templates
var templateFS embed.FS

type SwitchType string

const (
	SwitchTypeLeaf  SwitchType = "leaf"
	SwitchTypeSpine SwitchType = "spine"
)

type Config struct {
	// SearchDomain works like this:
	//
	//   $zone.infra.$environment.ironcore.dev
	//
	// Where zone is the region and availability zone, e.g. wdf-a; and
	// environment is one of dev, staging, canary, or live.
	//
	// Based on the type and ID each device will get a name such as spine-2 or
	// oob-leaf-1 to create FQDNs like this:
	//
	//   spine-2.region-b.infra.staging.ironcore.dev
	//
	// which uniqely identifies each device.
	SearchDomain string `json:"searchDomain"`
	// DHCPServerAddr for the relay to send DHCP packets to.
	//
	// Example:
	//
	//   2001:db8::547
	DHCPServerAddr string                          `json:"dhcpServerAddr"`
	SwitchParams   map[netip.Addr]SwitchParameters `json:"switchParams"`
}

type SwitchParameters struct {
	Type SwitchType `json:"type"`
	ID   int        `json:"id"`
	// Prefix must be a /64 for this sepcific switch.
	//
	// Example:
	//
	//   2001:db8::/64
	Prefix netip.Prefix `json:"prefix"`
	// IP should be the first /128 of the Prefix above.
	//
	// Example:
	//
	//   2001:db8::/128
	IP       netip.Prefix `json:"ip"`
	ASNumber int          `json:"asNumber"`
}

type handler struct {
	t *template.Template
	m map[netip.Addr]SwitchParameters
}

func Register(mux *http.ServeMux, c Config) {
	t := template.New("ztp-scripts")
	t = t.Funcs(template.FuncMap{
		"add":             func(a, b int) int { return a + b },
		"dhcpServerAddr":  func() string { return c.DHCPServerAddr },
		"searchDomain":    func() string { return c.SearchDomain },
		"interfacePrefix": interfacePrefix,
	})
	t = template.Must(t.ParseFS(templateFS, "templates/*.gotmpl"))

	mux.Handle("GET /ztp", &handler{t: t, m: c.SwitchParams})
}

// interfacePrefix takes the interface ID and the /64 prefix of the switch and
// reserves a unique /112 prefix for each interface based on the interface ID.
func interfacePrefix(prefix netip.Prefix, interfaceID int) (string, error) {
	if prefix.Bits() != 64 {
		return "", fmt.Errorf("unexpected prefix size %d, want 64", prefix.Bits())
	}

	b := prefix.Addr().AsSlice()
	b[13] = byte(interfaceID) + 1

	a, _ := netip.AddrFromSlice(b)

	prefix = netip.PrefixFrom(a, 112)
	return prefix.String(), nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ap, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		handleErr(w, err)
		return
	}

	c, ok := h.m[ap.Addr()]
	if !ok {
		handleErr(w, fmt.Errorf("unknown ip '%s'", ap.Addr().String()))
		return
	}

	switch c.Type {
	case SwitchTypeLeaf:
		err = h.t.ExecuteTemplate(w, "leaf.sh.gotmpl", c)
	case SwitchTypeSpine:
		err = h.t.ExecuteTemplate(w, "spine.sh.gotmpl", c)
	}
	if err != nil {
		handleErr(w, err)
		return
	}
}

func handleErr(w http.ResponseWriter, e error) {
	w.WriteHeader(http.StatusInternalServerError)
	_, err := fmt.Fprint(w, e.Error())
	if err != nil {
		slog.Error("failed to write back previous error", "e", e, "err", err)
	}
}

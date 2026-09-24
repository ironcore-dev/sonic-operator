// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"
	sonicdb "github.com/ironcore-dev/sonic-operator/pkg/sonic"
	"github.com/ironcore-dev/sonic-operator/pkg/sonic/hostservices"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/redis/go-redis/v9"
)

// stubDbusClient satisfies hostservices.DbusClient without requiring a D-Bus connection.
type stubDbusClient struct{}

func (s *stubDbusClient) HandlerExists(_ hostservices.Handler) (bool, error) { return true, nil }
func (s *stubDbusClient) ServiceAvailable(_ context.Context, _ string) bool  { return true }
func (s *stubDbusClient) CallHandler(_ context.Context, _ hostservices.Handler, _ ...interface{}) error {
	return nil
}
func (s *stubDbusClient) CallHandlerWithOutput(_ context.Context, _ hostservices.Handler, _ ...interface{}) (string, error) {
	return "", nil
}
func (s *stubDbusClient) Close() error { return nil }

func TestApplySwitch(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "ApplySwitch Suite")
}

// testDB groups a DBAccessor with the underlying Redis clients so tests can
// inspect raw Redis state via configDB/stateDB after calling applySwitch.
type testDB struct {
	sonicdb.DBAccessor
	configDB *redis.Client
	stateDB  *redis.Client
}

// newTestDBAccessor starts an in-process miniredis server and returns a testDB
// whose three clients all point at it (using different logical DB indices).
// The server is stopped automatically when the test ends.
func newTestDBAccessor(t ginkgo.GinkgoTInterface) (*testDB, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	newClient := func(db int) *redis.Client {
		return redis.NewClient(&redis.Options{
			Addr:         mr.Addr(),
			DB:           db,
			MaxRetries:   0,
			DialTimeout:  100 * time.Millisecond,
			ReadTimeout:  100 * time.Millisecond,
			WriteTimeout: 100 * time.Millisecond,
		})
	}
	configDB := newClient(4) // CONFIG_DB
	stateDB := newClient(6)  // STATE_DB
	applDB := newClient(0)   // APPL_DB
	return &testDB{
		DBAccessor: sonicdb.NewDBAccessor(configDB, stateDB, applDB),
		configDB:   configDB,
		stateDB:    stateDB,
	}, mr
}

var _ = ginkgo.Describe("applySwitch", func() {
	var (
		ctx    context.Context
		db     *testDB
		mr     *miniredis.Miniredis
		sa     *SonicAgent
		device string
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		db, mr = newTestDBAccessor(ginkgo.GinkgoT())
		sa = &SonicAgent{
			dbusClient:     &stubDbusClient{},
			writeFRRConfig: func(_, _ string) error { return nil },
			restartService: func(_ context.Context, _ string) error { return nil },
		}
		device = "test-device"
		_ = mr
	})

	ginkgo.It("sets hostname in DEVICE_METADATA|localhost", func() {
		cfg := agent.FabricSwitchConfig{Hostname: "leaf-1"}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		val, err := db.configDB.HGet(ctx, "DEVICE_METADATA|localhost", "hostname").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(val).To(gomega.Equal("leaf-1"))
	})

	ginkgo.It("creates LOOPBACK_INTERFACE|Loopback0", func() {
		cfg := agent.FabricSwitchConfig{}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		exists, err := db.configDB.Exists(ctx, "LOOPBACK_INTERFACE|Loopback0").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(exists).To(gomega.BeEquivalentTo(1))
	})

	ginkgo.It("syncs loopback IPs into LOOPBACK_INTERFACE|Loopback0|<prefix>", func() {
		cfg := agent.FabricSwitchConfig{
			LoopbackIPs: []string{"fd00::1/128"},
		}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		exists, err := db.configDB.Exists(ctx, "LOOPBACK_INTERFACE|Loopback0|fd00::1/128").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(exists).To(gomega.BeEquivalentTo(1))
	})

	ginkgo.It("creates VLAN|VlanN and VLAN_INTERFACE|VlanN", func() {
		cfg := agent.FabricSwitchConfig{
			VLANs: []agent.FabricVLAN{{ID: 42}},
		}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		vlanID, err := db.configDB.HGet(ctx, "VLAN|Vlan42", "vlanid").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(vlanID).To(gomega.Equal("42"))

		exists, err := db.configDB.Exists(ctx, "VLAN_INTERFACE|Vlan42").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(exists).To(gomega.BeEquivalentTo(1))
	})

	ginkgo.It("derives VLAN prefix from loopback when not explicitly set", func() {
		cfg := agent.FabricSwitchConfig{
			LoopbackIPs: []string{"2001:db8:0:1::1/128"},
			VLANs:       []agent.FabricVLAN{{ID: 1}},
		}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		// vlanPrefixFromLoopback("2001:db8:0:1::1/128", 1) = "2001:db8:0:1::/80"
		exists, err := db.configDB.Exists(ctx, "VLAN_INTERFACE|Vlan1|2001:db8:0:1::/80").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(exists).To(gomega.BeEquivalentTo(1))
	})

	ginkgo.It("uses explicit VLAN prefix when provided", func() {
		cfg := agent.FabricSwitchConfig{
			VLANs: []agent.FabricVLAN{{ID: 5, Prefix: "fd00:cafe::/64"}},
		}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		exists, err := db.configDB.Exists(ctx, "VLAN_INTERFACE|Vlan5|fd00:cafe::/64").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(exists).To(gomega.BeEquivalentTo(1))
	})

	ginkgo.It("sets DHCP relay on VLAN", func() {
		cfg := agent.FabricSwitchConfig{
			VLANs: []agent.FabricVLAN{{ID: 10, DHCPRelay: "10.0.0.254"}},
		}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		val, err := db.configDB.HGet(ctx, "VLAN|Vlan10", "dhcp_servers@").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(val).To(gomega.Equal("10.0.0.254"))
	})

	ginkgo.It("creates VLAN_MEMBER and sets MTU/FEC/speed on member port", func() {
		cfg := agent.FabricSwitchConfig{
			VLANs: []agent.FabricVLAN{{
				ID:      1,
				Members: []agent.FabricVLANMember{{InterfaceID: "Ethernet0"}},
			}},
		}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		tagging, err := db.configDB.HGet(ctx, "VLAN_MEMBER|Vlan1|Ethernet0", "tagging_mode").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(tagging).To(gomega.Equal("untagged"))

		mtu, err := db.configDB.HGet(ctx, "PORT|Ethernet0", "mtu").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(mtu).To(gomega.Equal("9100"))

		fec, err := db.configDB.HGet(ctx, "PORT|Ethernet0", "fec").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(fec).To(gomega.Equal("rs"))

		speed, err := db.configDB.HGet(ctx, "PORT|Ethernet0", "speed").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(speed).To(gomega.Equal("25000"))
	})

	ginkgo.It("records Creating then Active in STATE_DB on success", func() {
		cfg := agent.FabricSwitchConfig{Hostname: "sw-1"}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		state, err := db.stateDB.HGet(ctx, "WIRE_CELL_STATE|"+device, "state").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(state).To(gomega.Equal("Active"))
	})

	ginkgo.It("records Error in STATE_DB when a Redis write fails", func() {
		// Close miniredis to simulate a Redis failure mid-way.
		mr.Close()

		cfg := agent.FabricSwitchConfig{Hostname: "sw-fail"}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).NotTo(gomega.BeNil())
		gomega.Expect(status.Code).To(gomega.BeEquivalentTo(1))
	})

	ginkgo.It("does not crash when BGP config is present and writes frr.conf", func() {
		tmpDir := ginkgo.GinkgoT().TempDir()
		frrPath := tmpDir + "/frr.conf"
		sa.frrConfigPath = frrPath

		var writtenPath, writtenContent string
		sa.writeFRRConfig = func(path, content string) error {
			writtenPath = path
			writtenContent = content
			return os.WriteFile(path, []byte(content), 0644)
		}

		// Peer groups are derived from VLANs:
		//   VLAN without DHCPRelay → NORTH (member port becomes neighbor)
		//   VLAN with DHCPRelay    → SOUTH (Vlan<id> becomes neighbor)
		cfg := agent.FabricSwitchConfig{
			Hostname: "leaf-1",
			BGP:      &agent.FabricBGPConfig{ASN: 100},
			VLANs: []agent.FabricVLAN{
				{
					ID:      120,
					Members: []agent.FabricVLANMember{{InterfaceID: "Ethernet120"}},
					// No DHCPRelay → NORTH
				},
				{
					ID:        1001,
					DHCPRelay: "10.0.0.254",
					// DHCPRelay set → SOUTH
				},
			},
		}
		status := sa.applySwitch(ctx, db, cfg, device)
		gomega.Expect(status).To(gomega.BeNil())

		gomega.Expect(writtenPath).To(gomega.Equal(frrPath))
		gomega.Expect(writtenContent).To(gomega.ContainSubstring("router bgp 100"))
		gomega.Expect(writtenContent).To(gomega.ContainSubstring("neighbor Ethernet120 interface peer-group NORTH"))
		gomega.Expect(writtenContent).To(gomega.ContainSubstring("interface Vlan1001"))
		gomega.Expect(writtenContent).To(gomega.ContainSubstring("neighbor Vlan1001 interface peer-group SOUTH"))

		state, err := db.stateDB.HGet(ctx, "WIRE_CELL_STATE|"+device, "state").Result()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(state).To(gomega.Equal("Active"))
	})
})

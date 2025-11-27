// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"fmt"
	"sync"
	"time"

	errors "github.com/ironcore-dev/switch-operator/internal/agent/errors"
	agent "github.com/ironcore-dev/switch-operator/internal/agent/types"

	"github.com/redis/go-redis/v9"
	"github.com/vishvananda/netlink"
)

const (
	RedisDialTimeout     = 30 * time.Second
	RedisReadTimeout     = 5 * time.Second
	RedisWriteTimeout    = 5 * time.Second
	RedisPoolTimeout     = 10 * time.Second
	RedisMaxRetries      = 10
	RedisMinRetryBackoff = 500 * time.Millisecond
	RedisMaxRetryBackoff = 10 * time.Second
	RedisDefaultTimeout  = 5 * time.Second
)

type SonicAgent struct {
	redisAddr  string
	clientPool map[string]*redis.Client
	poolMutex  sync.RWMutex
}

func getRedisDBIDByName(name string) int {
	switch name {
	case "APPL_DB":
		return 0
	case "ASIC_DB":
		return 1
	case "COUNTERS_DB":
		return 2
	case "LOGLEVEL_DB":
		return 3
	case "CONFIG_DB":
		return 4
	case "PFC_WD_DB":
		return 5
	case "FLEX_COUNTER_DB":
		return 5
	case "STATE_DB":
		return 6
	case "SNMP_OVERLAY_DB":
		return 7
	case "RESTagent_DB":
		return 8
	case "GB_ASIC_DB":
		return 9
	case "GB_COUNTERS_DB":
		return 10
	case "GB_FLEX_COUNTER_DB":
		return 11
	case "APPL_STATE_DB":
		return 14
	default:
		return -1
	}
}

func NewSonicRedisAgent(redisAddr string) (*SonicAgent, error) {
	// Test connection first
	testClient := redis.NewClient(&redis.Options{
		Addr:            redisAddr,
		DB:              4, // Test with CONFIG_DB
		DialTimeout:     RedisDialTimeout,
		ReadTimeout:     RedisReadTimeout,
		WriteTimeout:    RedisWriteTimeout,
		PoolTimeout:     RedisPoolTimeout,
		MaxRetries:      RedisMaxRetries,
		MinRetryBackoff: RedisMinRetryBackoff,
		MaxRetryBackoff: RedisMaxRetryBackoff,
		DisableIndentity: true, // Disable identity/protocol checks to avoid warnings
	})

	if err := testClient.Ping(context.Background()).Err(); err != nil {
		if err := testClient.Close(); err != nil {
			return nil, fmt.Errorf("failed to close Redis client: %w", err)
		}
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	if err := testClient.Close(); err != nil {
		return nil, fmt.Errorf("failed to close Redis client: %w", err)
	}

	return &SonicAgent{
		redisAddr:  redisAddr,
		clientPool: make(map[string]*redis.Client),
		poolMutex:  sync.RWMutex{},
	}, nil
}

func (m *SonicAgent) Connect(dbName string) (*redis.Client, error) {
	m.poolMutex.RLock()
	if client, exists := m.clientPool[dbName]; exists {
		m.poolMutex.RUnlock()

		// Test if connection is still alive
		if err := client.Ping(context.Background()).Err(); err == nil {
			return client, nil
		}
	} else {
		m.poolMutex.RUnlock()
	}

	// Need to create new client (write lock)
	m.poolMutex.Lock()
	defer m.poolMutex.Unlock()

	// Double-check in case another goroutine created it
	if client, exists := m.clientPool[dbName]; exists {
		if err := client.Ping(context.Background()).Err(); err == nil {
			return client, nil
		}
		// Close the dead connection
		if err := client.Close(); err != nil {
			return nil, fmt.Errorf("failed to close Redis client: %w", err)
		}
		delete(m.clientPool, dbName)
	}

	// Create new client
	dbID := getRedisDBIDByName(dbName)
	if dbID == -1 {
		return nil, fmt.Errorf("unknown database name: %s", dbName)
	}

	client := redis.NewClient(&redis.Options{
		Addr:         m.redisAddr,
		DB:           dbID,
		DialTimeout:  RedisDialTimeout,
		ReadTimeout:  RedisReadTimeout,
		WriteTimeout: RedisWriteTimeout,
		PoolTimeout:  RedisPoolTimeout,
		MaxRetries:   RedisMaxRetries,

		// Connection pool settings
		PoolSize:     10, // Maximum number of socket connections
		MinIdleConns: 2,  // Minimum idle connections
		MaxIdleConns: 5,  // Maximum idle connections

		// Connection lifecycle
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: 1 * time.Hour,
		
		DisableIndentity: true, // Disable identity/protocol checks to avoid warnings
	})

	// Test the new connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		if err := client.Close(); err != nil {
			return nil, fmt.Errorf("failed to close Redis client: %w", err)
		}
		return nil, fmt.Errorf("failed to connect to Redis database %s: %w", dbName, err)
	}

	m.clientPool[dbName] = client

	return client, nil
}

func (m *SonicAgent) GetDeviceInfo(ctx context.Context) (*agent.SwitchDevice, *agent.Status) {
	rdb, err := m.Connect("CONFIG_DB")
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to connect to Redis: %v", err))
	}

	const deviceKey = "DEVICE_METADATA|localhost"
	fields, err := rdb.HGetAll(ctx, deviceKey).Result()
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to get device info: %v", err))
	}

	mac, ok := fields["mac"]
	if !ok {
		return nil, errors.NewErrorStatus(errors.NOT_FOUND, "missing or invalid MAC address")
	}

	hwsku := fields["hwsku"]
	sonicOSVersion := fields["sonic_os_version"]
	asicType := fields["asic_type"]

	// If values are missing from Redis, try to get from sonic_version.yml
	if hwsku == "" || sonicOSVersion == "" || asicType == "" {
		if versionInfo, err := GetSonicVersionInfo(); err == nil {
			if hwsku == "" {
				hwsku = versionInfo["hwsku"]
			}
			if sonicOSVersion == "" {
				sonicOSVersion = versionInfo["sonic_os_version"]
			}
			if asicType == "" {
				asicType = versionInfo["asic_type"]
			}
		}
	}

	return &agent.SwitchDevice{
		TypeMeta: agent.TypeMeta{
			Kind: agent.DeviceKind,
		},
		LocalMacAddress: mac,
		Hwsku:           hwsku,
		SonicOSVersion:  sonicOSVersion,
		AsicType:        asicType,
		Readiness:       uint32(agent.StatusReady),
	}, nil
}

func (m *SonicAgent) ListInterfaces(ctx context.Context) (*agent.InterfaceList, *agent.Status) {
	configDB, err := m.Connect("CONFIG_DB")
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to connect to CONFIG_DB: %v", err))
	}

	// Connect to STATE_DB for operational status
	stateDB, err := m.Connect("STATE_DB")
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to connect to STATE_DB: %v", err))
	}
	// defer stateDB.Close()

	pattern := "PORT|*"
	keys, err := configDB.Keys(ctx, pattern).Result()

	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to obtain iface keys: %v", err))
	}

	interfaces := make([]agent.Interface, 0, len(keys))
	for _, key := range keys {
		var name string
		if _, err := fmt.Sscanf(key, "PORT|%s", &name); err != nil {
			return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to parse interface name from key %s: %v", key, err))
		}

		// Get operational status from STATE_DB
		stateKey := fmt.Sprintf("PORT_TABLE|%s", name)
		stateFields, err := stateDB.HGetAll(ctx, stateKey).Result()
		if err != nil {
			// If state info is not available, use default values
			stateFields = make(map[string]string)
		}

		// Determine operational status
		operStatus := agent.StatusDown
		if stateFields["netdev_oper_status"] == "up" {
			operStatus = agent.StatusUp
		}

		adminStatus := agent.StatusDown
		if stateFields["admin_status"] == "up" {
			adminStatus = agent.StatusUp
		}

		// Use device MAC as interface MAC (common in SONiC)
		link, err := netlink.LinkByName(name)
		if err != nil {
			return nil, agent.NewErrorStatus(errors.NOT_FOUND, fmt.Sprintf("failed to get interface %s: %v", name, err))
		}

		mac := link.Attrs().HardwareAddr
		if mac == nil {
			return nil, agent.NewErrorStatus(errors.NOT_FOUND, fmt.Sprintf("no MAC address found for interface %s", name))
		}

		iface := agent.Interface{
			TypeMeta: agent.TypeMeta{
				Kind: agent.InterfaceKind,
			},
			Name:            name,
			MacAddress:      mac.String(),
			OperationStatus: operStatus,
			AdminStatus:     adminStatus,
		}
		interfaces = append(interfaces, iface)
	}

	return &agent.InterfaceList{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceListKind,
		},
		Items:  interfaces,
		Status: agent.Status{Code: 0, Message: "ok"},
	}, nil
}

func (m *SonicAgent) SetInterfaceAdminStatus(ctx context.Context, iface *agent.Interface) (*agent.Interface, *agent.Status) {
	configDB, err := m.Connect("CONFIG_DB")
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to connect to CONFIG_DB: %v", err))
	}

	// Validate interface name
	if iface.Name == "" {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, "interface name cannot be empty")
	}

	portKey := fmt.Sprintf("PORT|%s", iface.Name)

	// store the current admin status for rollback
	fields, err := configDB.HGetAll(ctx, portKey).Result()
	if err != nil {
		return nil, errors.NewErrorStatus(errors.REDIS_KEY_CHECK_FAIL, fmt.Sprintf("failed to get current admin status: %v", err))
	}
	currentAdminStatus := fields["admin_status"]

	// Set admin status in CONFIG_DB
	adminStatusStr := string(iface.AdminStatus)
	err = configDB.HSet(ctx, portKey, "admin_status", adminStatusStr).Err()
	if err != nil {
		return nil, errors.NewErrorStatus(errors.REDIS_HSET_FAIL, fmt.Sprintf("failed to set admin status: %v", err))
	}

	// Verify the interface exists by checking if we can get its current state
	exists, err := configDB.Exists(ctx, portKey).Result()
	if err != nil {
		return nil, errors.NewErrorStatus(errors.REDIS_KEY_CHECK_FAIL, fmt.Sprintf("failed to verify interface existence: %v", err))
	}
	if exists == 0 {
		return nil, errors.NewErrorStatus(errors.NOT_FOUND, fmt.Sprintf("interface %s not found", iface.Name))
	}

	time.Sleep(1000 * time.Millisecond)

	// Get updated interface status from STATE_DB
	stateDB, err := m.Connect("STATE_DB")
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to connect to STATE_DB: %v", err))
	}

	stateKey := fmt.Sprintf("PORT_TABLE|%s", iface.Name)
	stateFields, err := stateDB.HGetAll(ctx, stateKey).Result()
	if err != nil {
		// rollback admin status
		err = configDB.HSet(ctx, portKey, "admin_status", currentAdminStatus).Err()
		if err != nil {
			return nil, errors.NewErrorStatus(errors.REDIS_HSET_FAIL, fmt.Sprintf("failed to rollback admin status: %v", err))
		}
		return nil, errors.NewErrorStatus(errors.REDIS_KEY_CHECK_FAIL, fmt.Sprintf("failed to get state info: %v", err))
	}


	
	// get the newest operational status
	operStatus := agent.StatusDown
	if stateFields["netdev_oper_status"] == "up" {
		operStatus = agent.StatusUp
	}

	// Return updated interface
	updatedIface := *iface
	updatedIface.OperationStatus = operStatus

	return &updatedIface, nil
}

func (m *SonicAgent) GetInterface(ctx context.Context, iface *agent.Interface) (*agent.Interface, *agent.Status) {
	// Validate input
	if iface == nil || iface.Name == "" {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, "interface name cannot be empty")
	}

	configDB, err := m.Connect("CONFIG_DB")
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to connect to CONFIG_DB: %v", err))
	}

	// Connect to STATE_DB for operational status
	stateDB, err := m.Connect("STATE_DB")
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to connect to STATE_DB: %v", err))
	}

	// Check if interface exists in CONFIG_DB
	portKey := fmt.Sprintf("PORT|%s", iface.Name)
	exists, err := configDB.Exists(ctx, portKey).Result()
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to check interface existence: %v", err))
	}
	if exists == 0 {
		return nil, errors.NewErrorStatus(errors.NOT_FOUND, fmt.Sprintf("interface %s not found", iface.Name))
	}

	// Get operational status from STATE_DB
	stateKey := fmt.Sprintf("PORT_TABLE|%s", iface.Name)
	stateFields, err := stateDB.HGetAll(ctx, stateKey).Result()
	if err != nil {
		// If state info is not available, use default values
		stateFields = make(map[string]string)
	}

	// Determine operational status
	operStatus := agent.StatusDown
	if stateFields["netdev_oper_status"] == "up" {
		operStatus = agent.StatusUp
	}

	adminStatus := agent.StatusDown
	if stateFields["admin_status"] == "up" {
		adminStatus = agent.StatusUp
	}

	// Get interface MAC address using netlink
	link, err := netlink.LinkByName(iface.Name)
	if err != nil {
		return nil, errors.NewErrorStatus(errors.NOT_FOUND, fmt.Sprintf("failed to get interface %s: %v", iface.Name, err))
	}

	mac := link.Attrs().HardwareAddr
	if mac == nil {
		return nil, errors.NewErrorStatus(errors.NOT_FOUND, fmt.Sprintf("no MAC address found for interface %s", iface.Name))
	}

	resultInterface := &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name:            iface.Name,
		MacAddress:      mac.String(),
		OperationStatus: operStatus,
		AdminStatus:     adminStatus,
		Status:          agent.Status{Code: 0, Message: "ok"},
	}

	return resultInterface, nil
}

func (m *SonicAgent) GetInterfaceNeighbor(ctx context.Context, iface *agent.Interface) (*agent.InterfaceNeighbor, *agent.Status) {
	if iface == nil || iface.Name == "" {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, "interface name cannot be empty")
	}

	applDB, err := m.Connect("APPL_DB")
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to connect to APPL_DB: %v", err))
	}

	lldpKey := fmt.Sprintf("LLDP_ENTRY_TABLE:%s", iface.Name)

	// Check if LLDP entry exists for this interface
	exists, err := applDB.Exists(ctx, lldpKey).Result()
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to check LLDP entry existence: %v", err))
	}
	if exists == 0 {
		return nil, errors.NewErrorStatus(errors.NOT_FOUND, fmt.Sprintf("no LLDP neighbor found for interface %s", iface.Name))
	}

	// Get all LLDP fields
	lldpFields, err := applDB.HGetAll(ctx, lldpKey).Result()
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to get LLDP entry: %v", err))
	}

	// MacAddress from lldp_rem_chassis_id (when chassis_id_subtype is 4 - MAC address)
	macAddress := lldpFields["lldp_rem_chassis_id"]

	// SystemName from lldp_rem_sys_name
	systemName := lldpFields["lldp_rem_sys_name"]

	// Handle (remote interface name) from lldp_rem_port_desc
	// Note: lldp_rem_port_id contains "Eth5(Port5)" format, lldp_rem_port_desc contains "Ethernet16"
	handle := lldpFields["lldp_rem_port_desc"]
	if handle == "" {
		// Fallback to lldp_rem_port_id if port_desc is not available
		handle = lldpFields["lldp_rem_port_id"]
	}

	// Validate that we have the essential information
	if macAddress == "" || systemName == "" {
		return nil, errors.NewErrorStatus(errors.NOT_FOUND, fmt.Sprintf("incomplete LLDP information for interface %s", iface.Name))
	}

	neighbor := &agent.InterfaceNeighbor{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceNeighborKind,
		},
		Name:       iface.Name, // Interface name of yourself
		MacAddress: macAddress,
		SystemName: systemName,
		Handle:     handle, // Remote interface name
		Status:     agent.Status{Code: 0, Message: "ok"},
	}

	return neighbor, nil
}

func (m *SonicAgent) ListPorts(ctx context.Context) (*agent.PortList, *agent.Status) {
	// Connect to APPL_DB (table 0)
	applDB, err := m.Connect("APPL_DB")
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to connect to APPL_DB: %v", err))
	}

	// List keys starting with PORT_TABLE
	pattern := "PORT_TABLE:*"
	keys, err := applDB.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, errors.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("failed to obtain PORT_TABLE keys: %v", err))
	}

	ports := make([]agent.Port, 0)
	for _, key := range keys {
		var portName string
		if _, err := fmt.Sscanf(key, "PORT_TABLE:%s", &portName); err != nil {
			continue // Skip malformed keys
		}

		// Get the port configuration
		fields, err := applDB.HGetAll(ctx, key).Result()
		if err != nil {
			continue // Skip if we can't get the fields
		}

		// Check if this represents a physical port by examining the "parent_port" field
		// If parent_port equals the port name itself, it's a physical port
		parentPort, exists := fields["parent_port"]
		if !exists || parentPort != portName {
			continue // Skip non-physical ports (sub-interfaces, VLANs, etc.)
		}

		// Get alias if available
		alias := fields["alias"]
		if alias == "" {
			alias = portName // Use port name as alias if not specified
		}

		port := agent.Port{
			TypeMeta: agent.TypeMeta{
				Kind: agent.PortKind,
			},
			Name:   portName,
			Alias:  alias,
			Status: agent.Status{Code: 0, Message: "ok"},
		}
		ports = append(ports, port)
	}

	return &agent.PortList{
		TypeMeta: agent.TypeMeta{
			Kind: agent.PortListKind,
		},
		Items:  ports,
		Status: agent.Status{Code: 0, Message: "ok"},
	}, nil
}

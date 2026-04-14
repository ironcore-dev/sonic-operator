// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package hostservices

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"sigs.k8s.io/yaml"
)

//go:embed modules
var modulesFS embed.FS

// Arg represents a dbus method argument.
type Arg struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// Handler maps a logical operation name to a dbus interface/method call.
type Handler struct {
	Name        string `json:"name"`
	ServiceName string `json:"serviceName"`
	ObjectPath  string `json:"objectPath"`
	Method      string `json:"method"`
	// Builtin indicates the handler is shipped with SONiC and expected to be
	// present on the system. false means it was custom-installed.
	Builtin bool  `json:"builtin"`
	Args    []Arg `json:"args"`
	// IgnoreCodes lists application-level return codes that should be treated as
	// success. For example, systemctl restart returns -15 (SIGTERM) when it
	// kills the service being restarted, which is expected behaviour.
	IgnoreCodes []int32 `json:"ignore_codes"`
}

// DbusMapping holds all handlers for a given module variant.
type DbusMapping struct {
	HostServicesPackagesDir string    `json:"hostservices_packages_dir"`
	Handlers                []Handler `json:"handlers"`
}

// DbusClient abstracts D-Bus operations for testability.
type DbusClient interface {
	HandlerExists(h Handler) (bool, error)
	ServiceAvailable(ctx context.Context, serviceName string) bool
	CallHandler(ctx context.Context, h Handler, args ...interface{}) error
	Close() error
}

// SystemDbusClient is the real D-Bus implementation.
type SystemDbusClient struct {
	conn *dbus.Conn
}

func NewSystemDbusClient() (*SystemDbusClient, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect to system bus: %w", err)
	}
	return &SystemDbusClient{conn: conn}, nil
}

func (c *SystemDbusClient) HandlerExists(h Handler) (bool, error) {
	node, err := introspect.Call(c.conn.Object(h.ServiceName, dbus.ObjectPath(h.ObjectPath)))
	if err != nil {
		return false, fmt.Errorf("introspect %s: %w", h.ObjectPath, err)
	}
	for _, iface := range node.Interfaces {
		if iface.Name != h.ServiceName {
			continue
		}
		for _, method := range iface.Methods {
			if method.Name == h.Method {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *SystemDbusClient) ServiceAvailable(ctx context.Context, serviceName string) bool {
	var hasOwner bool
	err := c.conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.NameHasOwner", 0, serviceName).Store(&hasOwner)
	if err != nil {
		return false
	}
	return hasOwner
}

func (c *SystemDbusClient) CallHandler(ctx context.Context, h Handler, args ...any) error {
	obj := c.conn.Object(h.ServiceName, dbus.ObjectPath(h.ObjectPath))
	call := obj.CallWithContext(ctx, h.Method, 0, args...)
	if call.Err != nil {
		return call.Err
	}
	if len(call.Body) >= 2 {
		rc, _ := call.Body[0].(int32)
		msg, _ := call.Body[1].(string)
		if rc != 0 {
			for _, ignored := range h.IgnoreCodes {
				if rc == ignored {
					return nil
				}
			}
			return fmt.Errorf("handler %s returned error (code %d): %s", h.Name, rc, msg)
		}
	}
	return nil
}

func (c *SystemDbusClient) Close() error {
	return c.conn.Close()
}

// Modules contains the loaded dbus mappings keyed by module name (e.g. "default", "edgecore").
var Modules map[string]DbusMapping

func init() {
	loaded, err := loadModules()
	if err != nil {
		panic(fmt.Sprintf("hostservices: failed to load dbus mappings: %v", err))
	}
	Modules = loaded
}

// ModuleFile returns the raw bytes of a file inside the embedded modules directory.
// module is e.g. "default" or "edgecore", filename is e.g. "onie.py".
func ModuleFile(module, filename string) ([]byte, error) {
	return fs.ReadFile(modulesFS, "modules/"+module+"/"+filename)
}

func loadModules() (map[string]DbusMapping, error) {
	result := make(map[string]DbusMapping)

	entries, err := modulesFS.ReadDir("modules")
	if err != nil {
		return nil, fmt.Errorf("read modules dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		log.Printf("hostservices: loading module %q", name)
		data, err := fs.ReadFile(modulesFS, "modules/"+name+"/dbus_mapping.yaml")
		if err != nil {
			return nil, fmt.Errorf("read dbus_mapping.yaml for module %q: %w", name, err)
		}

		var mapping DbusMapping
		if err := yaml.Unmarshal(data, &mapping); err != nil {
			return nil, fmt.Errorf("parse dbus_mapping.yaml for module %q: %w", name, err)
		}

		result[name] = mapping
		log.Printf("hostservices: loaded module %q with %d handlers", name, len(mapping.Handlers))
	}

	return result, nil
}

func HostServicesCompatibilityCheck(ctx context.Context, client DbusClient, profile string) ([]string, []error) {
	// Check if sonic-hostservice is running before probing individual handlers.
	// After a reboot the agent may start before sonic-hostservice, so all
	// introspection calls would fail and the agent would incorrectly try to
	// install modules.
	firstHandler := Modules[profile].Handlers[0]
	if !client.ServiceAvailable(ctx, firstHandler.ServiceName) {
		return nil, []error{fmt.Errorf("host service %q not yet available on D-Bus, sonic-hostservice may not be running", firstHandler.ServiceName)}
	}

	var compatible []string
	var errors []error

	for i, handler := range Modules[profile].Handlers {
		ok, err := client.HandlerExists(handler)
		if err != nil {
			errors = append(errors, fmt.Errorf("check handler %d (%s) [service=%s path=%s method=%s args=%v]: %w", i, handler.Name, handler.ServiceName, handler.ObjectPath, handler.Method, handler.Args, err))
			continue
		}
		if !ok {
			errors = append(errors, fmt.Errorf("handler %d (%s) not found on system D-Bus [service=%s path=%s method=%s args=%v]", i, handler.Name, handler.ServiceName, handler.ObjectPath, handler.Method, handler.Args))

		} else {
			compatible = append(compatible, handler.Name)
		}
	}
	return compatible, errors
}

func InstallHostServiceModule(client DbusClient, osProfile string, dbusModule string) error {
	log.Printf("InstallHostServiceModule:Installing host service module %q for dbus interface %q", osProfile, dbusModule)

	var objectPath string
	var pyFile string
	switch dbusModule {
	case "onie":
		objectPath = "/org/SONiC/HostService/onie"
		pyFile = "onie.py"
	case "reboot":
		objectPath = "/org/SONiC/HostService/reboot"
		pyFile = "reboot.py"
	default:
		return fmt.Errorf("unsupported module: %s", dbusModule)
	}

	var handler *Handler
	for _, h := range Modules[osProfile].Handlers {
		if h.ObjectPath == objectPath {
			handler = &h
			break
		}
	}
	if handler == nil {
		return fmt.Errorf("handler for %s module not found in dbus mapping", dbusModule)
	}

	exists, err := client.HandlerExists(*handler)
	if exists && err == nil {
		log.Printf("Host service module %q for dbus interface %q is already installed", osProfile, dbusModule)
		return nil // already installed
	}

	if handler.Builtin {
		return fmt.Errorf("handler %q is marked as builtin for profile %q but not found on system D-Bus", dbusModule, osProfile)
	}

	dir := Modules[osProfile].HostServicesPackagesDir
	if dir == "" {
		return fmt.Errorf("hostservices_packages_dir not defined for module %q", osProfile)
	}
	if !strings.HasPrefix(dir, "/") {
		return fmt.Errorf("hostservices_packages_dir must be an absolute path: %s", dir)
	}
	// check if the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("hostservices_packages_dir does not exist: %s", dir)
	}

	data, err := ModuleFile(osProfile, pyFile)
	if err != nil {
		return fmt.Errorf("read module file: %w", err)
	}
	targetPath := path.Join(dir, pyFile)
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return fmt.Errorf("write module file to target location: %w", err)
	}
	// edgecores sonic-hostservice-server has autodiscovery of host modules
	// however, newer versions of SONiC need to register modules into source files for them to be picked up. To handle both cases, we can trigger a reload of the sonic-hostservices systemd service which should cause it to pick up the new module whether it relies on autodiscovery or static registration.
	if osProfile == "default" {
		serverScript := "/usr/local/bin/sonic-host-server"
		content, err := os.ReadFile(serverScript)
		if err != nil {
			return fmt.Errorf("read server script: %w", err)
		}
		contentStr := string(content)
		// Add import if not present
		if !strings.Contains(contentStr, ", "+dbusModule) {
			contentStr = strings.Replace(contentStr, "from host_modules import", "from host_modules import "+dbusModule+", ", 1)
		}
		// Add registration if not present
		if !strings.Contains(contentStr, "'"+dbusModule+"':") {
			configLine := "'config': config_engine.Config('config'),"
			className := strings.ToUpper(dbusModule[:1]) + dbusModule[1:]
			insertAfter := configLine + "\n        '" + dbusModule + "': " + dbusModule + "." + className + "('" + dbusModule + "'),"
			contentStr = strings.Replace(contentStr, configLine, insertAfter, 1)
		}
		// Write back
		err = os.WriteFile(serverScript, []byte(contentStr), 0644)
		if err != nil {
			return fmt.Errorf("write server script: %w", err)
		}

		// Patch systemd_service.py to allow restarting sonic-hostservice
		systemdServiceScript := "/usr/local/lib/python3.11/dist-packages/host_modules/systemd_service.py"
		sdContent, err := os.ReadFile(systemdServiceScript)
		if err != nil {
			return fmt.Errorf("read systemd_service script: %w", err)
		}
		sdContentStr := string(sdContent)
		if !strings.Contains(sdContentStr, "'sonic-hostservice'") {
			sdContentStr = strings.Replace(sdContentStr, "ALLOWED_SERVICES = ['snmp',", "ALLOWED_SERVICES = ['sonic-hostservice', 'snmp',", 1)
			if err := os.WriteFile(systemdServiceScript, []byte(sdContentStr), 0644); err != nil {
				return fmt.Errorf("write systemd_service script: %w", err)
			}
		}
	}
	// TODO: restart systemd service for sonic-hostservices
	return nil
}

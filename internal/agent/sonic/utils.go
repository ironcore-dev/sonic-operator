// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"fmt"
	"os"
	"strings"

	agent "github.com/ironcore-dev/switch-operator/internal/agent/types"
)

func GetSonicVersionInfo() (map[string]string, error) {
	info := make(map[string]string)

	content, err := os.ReadFile("/etc/sonic/sonic_version.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read sonic_version.yml: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "---") || line == "" {
			continue
		}

		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Remove quotes if present
				value = strings.Trim(value, `'"`)
				info[key] = value
			}
		}
	}

	return info, nil
}

func ConvertAdminStatusToStr(adminStatus uint32) string {

	if adminStatus == uint32(agent.StatusUp) {
		return "up"
	} else {
		return "down"
	}
}

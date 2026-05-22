// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

func GetSonicVersionInfo() (map[string]string, error) {
	content, err := os.ReadFile("/etc/sonic/sonic_version.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read sonic_version.yml: %w", err)
	}

	info := make(map[string]string)
	if err := yaml.Unmarshal(content, &info); err != nil {
		return nil, fmt.Errorf("failed to parse sonic_version.yml: %w", err)
	}
	return info, nil
}

// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"fmt"
)

func (db *dbAccessor) SetHostname(ctx context.Context, hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname must not be empty")
	}
	if err := db.configDB.HSet(ctx, "DEVICE_METADATA|localhost", "hostname", hostname).Err(); err != nil {
		return fmt.Errorf("failed to set hostname: %w", err)
	}
	return nil
}

func (db *dbAccessor) GetDeviceMetadata(ctx context.Context) (map[string]string, error) {
	fields, err := db.configDB.HGetAll(ctx, "DEVICE_METADATA|localhost").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get device metadata: %w", err)
	}
	return fields, nil
}

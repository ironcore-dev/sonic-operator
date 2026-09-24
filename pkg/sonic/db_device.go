// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sonic

import (
	"context"
	"fmt"
)

const cellStateKeyPrefix = "WIRE_CELL_STATE|"

func (db *dbAccessor) SetHostname(ctx context.Context, hostname string) error {
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

func (db *dbAccessor) SetCellState(ctx context.Context, device, state, message string) error {
	if err := db.stateDB.HSet(ctx, cellStateKeyPrefix+device, "state", state, "message", message).Err(); err != nil {
		return fmt.Errorf("failed to set cell state for %s: %w", device, err)
	}
	return nil
}

func (db *dbAccessor) GetCellState(ctx context.Context, device string) (state, message string, err error) {
	fields, err := db.stateDB.HGetAll(ctx, cellStateKeyPrefix+device).Result()
	if err != nil {
		return "", "", fmt.Errorf("failed to get cell state for %s: %w", device, err)
	}
	return fields["state"], fields["message"], nil
}

func (db *dbAccessor) DeleteCellState(ctx context.Context, device string) error {
	if err := db.stateDB.Del(ctx, cellStateKeyPrefix+device).Err(); err != nil {
		return fmt.Errorf("failed to delete cell state for %s: %w", device, err)
	}
	return nil
}

// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConnector abstracts Redis database access for testability.
// *sonic.SonicAgent satisfies this interface via its Connect method.
type RedisConnector interface {
	Connect(dbName string) (*redis.Client, error)
}

// collectTimeout is the context timeout for Redis reads during a Prometheus scrape.
const collectTimeout = 8 * time.Second

// batchHGetAll fetches all hash fields for the given keys using a Redis pipeline.
// Returns a map from key to field-value map. Keys that fail are silently skipped.
func batchHGetAll(ctx context.Context, client *redis.Client, keys []string) map[string]map[string]string {
	if len(keys) == 0 {
		return nil
	}

	pipe := client.Pipeline()
	cmds := make(map[string]*redis.MapStringStringCmd, len(keys))
	for _, key := range keys {
		cmds[key] = pipe.HGetAll(ctx, key)
	}
	_, _ = pipe.Exec(ctx)

	result := make(map[string]map[string]string, len(keys))
	for key, cmd := range cmds {
		fields, err := cmd.Result()
		if err != nil || len(fields) == 0 {
			continue
		}
		result[key] = fields
	}
	return result
}

// extractKeySuffix returns the portion of a Redis key after the first separator
// ("|" or ":"). For example, "PORT_TABLE|Ethernet0" returns "Ethernet0".
func extractKeySuffix(key, sep string) string {
	_, after, ok := strings.Cut(key, sep)
	if !ok {
		return key
	}
	return after
}

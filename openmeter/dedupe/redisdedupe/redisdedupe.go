// Package redisdedupe implements event deduplication using Redis.
package redisdedupe

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/redis/go-redis/v9"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

type DedupeMode string

const (
	DedupeModeRawKey           DedupeMode = "rawkey"
	DedupeModeKeyHash          DedupeMode = "keyhash"
	DedupeModeKeyHashMigration DedupeMode = "keyhash-migration"
)

func (m DedupeMode) Validate() error {
	switch m {
	case DedupeModeRawKey, DedupeModeKeyHash, DedupeModeKeyHashMigration:
		return nil
	}
	return fmt.Errorf("invalid dedupe mode: %s", m)
}

// Deduplicator implements event deduplication using Redis.
type Deduplicator struct {
	Redis      *redis.Client
	Expiration time.Duration
	Mode       DedupeMode
}

// IsUnique checks if an event is unique AND adds it to the deduplication index.
func (d Deduplicator) IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error) {
	if d.Redis == nil {
		return false, errors.New("redis client not initialized")
	}

	item := dedupe.Item{
		Namespace: namespace,
		ID:        ev.ID(),
		Source:    ev.Source(),
	}

	switch d.Mode {
	case DedupeModeRawKey:
		return d.setKey(ctx, item.Key())
	case DedupeModeKeyHash:
		keyHash := getKeyHash(item.Key())
		return d.setKey(ctx, keyHash)
	case DedupeModeKeyHashMigration:
		keyHash := getKeyHash(item.Key())
		isUnique, err := d.setKey(ctx, keyHash)
		if err != nil {
			return false, err
		}

		// Migration to the new hashing format
		if isUnique {
			// We might have succeeded setting the key only because the key just exist in the old format
			// Let's check if the old key exists
			isSet, err := d.Redis.Exists(ctx, item.Key()).Result()
			if err != nil {
				return false, err
			}

			keyExists := isSet == 1

			return !keyExists, nil
		}
	}

	return false, fmt.Errorf("unknown status")
}

func (d Deduplicator) setKey(ctx context.Context, key string) (bool, error) {
	status, err := d.Redis.SetArgs(ctx, key, "", redis.SetArgs{
		TTL:  d.Expiration,
		Mode: "nx",
	}).Result()

	// This is an unusual API, see: https://github.com/redis/go-redis/blob/v9.0.5/commands_test.go#L1545
	// Redis returns redis.Nil
	if err != nil && err != redis.Nil {
		return false, err
	}

	// Key already existed before, so it's a duplicate
	if status == "" {
		return false, nil
	}
	// Key did not exist before, so it's unique
	if status == "OK" {
		return true, nil
	}

	return false, fmt.Errorf("unknown status")
}

// CheckUnique checks if the event is unique based on the key
func (d Deduplicator) CheckUnique(ctx context.Context, item dedupe.Item) (bool, error) {
	keysToCheck := make([]string, 0, 2)
	switch d.Mode {
	case DedupeModeRawKey:
		keysToCheck = append(keysToCheck, item.Key())
	case DedupeModeKeyHash:
		keysToCheck = append(keysToCheck, getKeyHash(item.Key()))
	case DedupeModeKeyHashMigration:
		keysToCheck = append(keysToCheck, item.Key(), getKeyHash(item.Key()))
	}

	isSet, err := d.Redis.Exists(ctx, keysToCheck...).Result()
	if err != nil {
		return false, err
	}

	return isSet == 0, nil
}

// Set sets events into redis
func (d Deduplicator) Set(ctx context.Context, items ...dedupe.Item) error {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		switch d.Mode {
		case DedupeModeRawKey:
			keys = append(keys, item.Key())
		case DedupeModeKeyHash, DedupeModeKeyHashMigration:
			keys = append(keys, getKeyHash(item.Key()))
		}
	}

	// We use a lua script to set multiple keys at once, this is more efficient than calling redis one by one
	err := setMultiple.Run(ctx, d.Redis, keys, d.Expiration.Seconds()).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to set multiple keys in redis: %w", err)
	}

	return nil
}

var setMultiple = redis.NewScript(`
local expiration = tonumber(ARGV[1])

for _, key in ipairs(KEYS) do
  redis.call("SET", key, "", "EX", expiration)
end
`)

// Close closes underlying redis client
func (d Deduplicator) Close() error {
	if d.Redis != nil {
		return d.Redis.Close()
	}

	return nil
}

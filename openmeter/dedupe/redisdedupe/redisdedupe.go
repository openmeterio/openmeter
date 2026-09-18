// Package redisdedupe implements event deduplication using Redis.
package redisdedupe

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
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
	_, unique, err := d.Claim(ctx, dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()})
	return unique, err
}

func (d Deduplicator) Claim(ctx context.Context, item dedupe.Item) (dedupe.Claim, bool, error) {
	if d.Redis == nil {
		return dedupe.Claim{}, false, errors.New("redis client not initialized")
	}
	claim := dedupe.Claim{Item: item, Token: uuid.NewString()}
	key := item.Key()
	if d.Mode != DedupeModeRawKey {
		key = GetKeyHash(key)
	}
	unique, err := d.setKey(ctx, key, claim.Token)
	if err != nil || !unique {
		return dedupe.Claim{}, unique, err
	}
	if d.Mode == DedupeModeKeyHashMigration {
		exists, err := d.Redis.Exists(ctx, item.Key()).Result()
		if err != nil {
			return dedupe.Claim{}, false, err
		}
		if exists == 1 {
			if err := d.Release(ctx, claim); err != nil {
				return dedupe.Claim{}, false, err
			}
			return dedupe.Claim{}, false, nil
		}
	}
	return claim, true, nil
}

func (d Deduplicator) setKey(ctx context.Context, key, value string) (bool, error) {
	status, err := d.Redis.SetArgs(ctx, key, value, redis.SetArgs{TTL: d.Expiration, Mode: "nx"}).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if status == "" {
		return false, nil
	}
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
		keysToCheck = append(keysToCheck, GetKeyHash(item.Key()))
	case DedupeModeKeyHashMigration:
		keysToCheck = append(keysToCheck, item.Key(), GetKeyHash(item.Key()))
	}

	isSet, err := d.Redis.Exists(ctx, keysToCheck...).Result()
	if err != nil {
		return false, err
	}

	return isSet == 0, nil
}

// Set sets events into redis
func (d Deduplicator) Set(ctx context.Context, items ...dedupe.Item) ([]dedupe.Item, error) {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		switch d.Mode {
		case DedupeModeRawKey:
			keys = append(keys, item.Key())
		case DedupeModeKeyHash, DedupeModeKeyHashMigration:
			keys = append(keys, GetKeyHash(item.Key()))
		}
	}

	cmds, err := d.Redis.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, key := range keys {
			_, err := pipe.SetArgs(ctx, key, "", redis.SetArgs{
				TTL:  d.Expiration,
				Mode: "NX",
			}).Result()
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to set multiple keys in redis: %w", err)
	}

	// Let's check if all the keys were created or if some of them already existed
	existingItems := []dedupe.Item{}
	for i, cmd := range cmds {
		item := items[i]
		if cmd.Err() != nil {
			if !errors.Is(cmd.Err(), redis.Nil) {
				return nil, fmt.Errorf("failed to set key %s in redis: %w", item.Key(), cmd.Err())
			}

			existingItems = append(existingItems, item)
		}
	}

	return existingItems, nil
}

// Release deletes a claim only if the stored ownership token still matches.
func (d Deduplicator) Release(ctx context.Context, claim dedupe.Claim) error {
	if d.Redis == nil {
		return errors.New("redis client not initialized")
	}
	key := claim.Item.Key()
	switch d.Mode {
	case DedupeModeRawKey:
	case DedupeModeKeyHash, DedupeModeKeyHashMigration:
		key = GetKeyHash(key)
	default:
		return fmt.Errorf("invalid dedupe mode: %s", d.Mode)
	}
	const compareAndDelete = `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) else return 0 end`
	if err := d.Redis.Eval(ctx, compareAndDelete, []string{key}, claim.Token).Err(); err != nil {
		return fmt.Errorf("failed to release claim in redis: %w", err)
	}
	return nil
}

// Close closes underlying redis client
func (d Deduplicator) Close() error {
	if d.Redis != nil {
		return d.Redis.Close()
	}

	return nil
}

var ErrNoDedupItems = errors.New("no dedup items provided")

func (d Deduplicator) CheckUniqueBatch(ctx context.Context, items []dedupe.Item) (dedupe.CheckUniqueBatchResult, error) {
	if len(items) == 0 {
		return dedupe.CheckUniqueBatchResult{}, ErrNoDedupItems
	}

	keysPerItem := 1
	if d.Mode == DedupeModeKeyHashMigration {
		keysPerItem = 2
	}

	keys := make([]string, 0, len(items)*keysPerItem)
	for _, item := range items {
		switch d.Mode {
		case DedupeModeRawKey:
			keys = append(keys, item.Key())
		case DedupeModeKeyHash:
			keys = append(keys, GetKeyHash(item.Key()))
		case DedupeModeKeyHashMigration:
			keys = append(keys, item.Key(), GetKeyHash(item.Key()))
		}
	}

	cmdResults, err := d.Redis.MGet(ctx, keys...).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return dedupe.CheckUniqueBatchResult{}, fmt.Errorf("failed to get multiple keys in redis: %w", err)
	}

	if len(cmdResults) != len(items)*keysPerItem {
		return dedupe.CheckUniqueBatchResult{}, fmt.Errorf("failed to get all keys in redis")
	}

	result := dedupe.CheckUniqueBatchResult{
		UniqueItems:           make(dedupe.ItemSet, len(items)),
		AlreadyProcessedItems: make(dedupe.ItemSet, len(items)),
	}

	for i, item := range items {
		unique := true
		for _, value := range cmdResults[i*keysPerItem : (i+1)*keysPerItem] {
			if value != nil {
				unique = false
				break
			}
		}

		if unique {
			result.UniqueItems[item] = struct{}{}
		} else {
			result.AlreadyProcessedItems[item] = struct{}{}
		}
	}

	return result, nil
}

// Package redisdedupe implements event deduplication using Redis.
package redisdedupe

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
	"github.com/zeebo/xxh3"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

const (
	defaultPreallocSize = 256
)

func NewHashDeduplicator(redis *redis.Client, expiration time.Duration) *HashDeduplicator {
	return &HashDeduplicator{
		Redis:      redis,
		Expiration: expiration,
	}
}

// HashDeduplicator implements event deduplication using Redis.
type HashDeduplicator struct {
	Redis      *redis.Client
	Expiration time.Duration
}

type HashKey struct {
	redisHashKey       string
	redisHashFieldName string
}

func (d HashDeduplicator) GetHashKey(i dedupe.Item) HashKey {
	hashBytes := xxh3.HashString128(i.Key()).Bytes()
	b64 := base64.RawURLEncoding.EncodeToString(hashBytes[1:])

	hashKeyPrefix := strconv.FormatInt(int64(hashBytes[0]), 16)

	return HashKey{
		// We use the first character of the hash too, so that we can have 256*2^32 items per namespace max,
		// as redis can only store 2^32 keys hash.
		redisHashKey:       i.Namespace + "/" + hashKeyPrefix,
		redisHashFieldName: b64,
	}
}

type HashKeyFieldWithItem struct {
	RedisHashFieldName string
	Item               dedupe.Item
}

type HashKeyFields map[string][]HashKeyFieldWithItem

func (d HashDeduplicator) GetHashKeysFields(items []dedupe.Item) HashKeyFields {
	out := make(HashKeyFields, defaultPreallocSize)
	for _, item := range items {

		hashKey := d.GetHashKey(item)

		if _, ok := out[hashKey.redisHashKey]; !ok {
			out[hashKey.redisHashKey] = make([]HashKeyFieldWithItem, 0, defaultPreallocSize)
		}

		out[hashKey.redisHashKey] = append(out[hashKey.redisHashKey], HashKeyFieldWithItem{
			RedisHashFieldName: hashKey.redisHashFieldName,
			Item:               item,
		})
	}
	return out
}

// IsUnique checks if an event is unique AND adds it to the deduplication index.
func (d HashDeduplicator) IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error) {
	if d.Redis == nil {
		return false, errors.New("redis client not initialized")
	}

	item := dedupe.Item{
		Namespace: namespace,
		ID:        ev.ID(),
		Source:    ev.Source(),
	}

	hashKey := d.GetHashKey(item)

	res, err := d.Redis.HSetEXWithArgs(ctx, hashKey.redisHashKey, &redis.HSetEXOptions{
		Condition:      redis.HSetEXFNX,
		ExpirationType: redis.HSetEXExpirationEX,
		ExpirationVal:  int64(d.Expiration.Seconds()),
	},
		hashKey.redisHashFieldName, "",
	).Result()
	if err != nil {
		return false, err
	}

	keyExists := res == 0

	return !keyExists, nil
}

// CheckUnique checks if the event is unique based on the key
func (d HashDeduplicator) CheckUnique(ctx context.Context, item dedupe.Item) (bool, error) {
	hashKeyWithField := d.GetHashKey(item)

	isSet, err := d.Redis.HExists(ctx, hashKeyWithField.redisHashKey, hashKeyWithField.redisHashFieldName).Result()
	if err != nil {
		return false, err
	}

	return !isSet, nil
}

// Set sets events into redis
func (d HashDeduplicator) Set(ctx context.Context, items ...dedupe.Item) ([]dedupe.Item, error) {
	cmds, err := d.Redis.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, item := range items {

			hashKey := d.GetHashKey(item)

			_, err := pipe.HSetEXWithArgs(ctx, hashKey.redisHashKey, &redis.HSetEXOptions{
				Condition:      redis.HSetEXFNX,
				ExpirationType: redis.HSetEXExpirationEX,
				ExpirationVal:  int64(d.Expiration.Seconds()),
			},
				hashKey.redisHashFieldName, "",
			).Result()
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
			continue
		}

		if intCmd, ok := cmd.(*redis.IntCmd); ok {
			res := intCmd.Val()
			if res == 0 {
				existingItems = append(existingItems, item)
			}
		} else {
			return nil, fmt.Errorf("unexpected response type for key %s: %T", item.Key(), cmd)
		}

	}

	return existingItems, nil
}

// Close closes underlying redis client
func (d HashDeduplicator) Close() error {
	if d.Redis != nil {
		return d.Redis.Close()
	}

	return nil
}

func (d HashDeduplicator) CheckUniqueBatch(ctx context.Context, items []dedupe.Item) (dedupe.CheckUniqueBatchResult, error) {
	hashKeysFields := d.GetHashKeysFields(items)

	itemsPerHMGet := make([][]HashKeyFieldWithItem, 0, len(hashKeysFields))

	cmds, err := d.Redis.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for hash, items := range hashKeysFields {
			_, err := pipe.HMGet(ctx, hash, lo.Map(items, func(item HashKeyFieldWithItem, _ int) string {
				return item.RedisHashFieldName
			})...).Result()

			itemsPerHMGet = append(itemsPerHMGet, items)

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil && !errors.Is(err, redis.Nil) {
		return dedupe.CheckUniqueBatchResult{}, fmt.Errorf("failed to get multiple keys in redis: %w", err)
	}

	if len(itemsPerHMGet) != len(cmds) {
		return dedupe.CheckUniqueBatchResult{}, fmt.Errorf("failed to get all keys in redis")
	}

	result := dedupe.CheckUniqueBatchResult{
		UniqueItems:           make(dedupe.ItemSet, len(items)),
		AlreadyProcessedItems: make(dedupe.ItemSet, len(items)),
	}

	for bucketIdx, cmdResult := range cmds {
		if cmdResult.Err() != nil {
			return dedupe.CheckUniqueBatchResult{}, fmt.Errorf("failed to get key %s in redis: %w", itemsPerHMGet[bucketIdx][0].Item.Key(), cmdResult.Err())
		}

		sliceCmd, ok := cmdResult.(*redis.SliceCmd)
		if !ok {
			return dedupe.CheckUniqueBatchResult{}, fmt.Errorf("unexpected response type for namespace %s: %T", itemsPerHMGet[bucketIdx][0].Item.Namespace, cmdResult)
		}

		for itemIdx, res := range sliceCmd.Val() {
			item := itemsPerHMGet[bucketIdx][itemIdx].Item

			if res == nil {
				result.UniqueItems[item] = struct{}{}
				continue
			}

			result.AlreadyProcessedItems[item] = struct{}{}
		}

	}

	return result, nil
}

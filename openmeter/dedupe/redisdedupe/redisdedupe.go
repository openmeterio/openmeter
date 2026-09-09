// Package redisdedupe implements event deduplication using Redis.
package redisdedupe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
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

type ClientFactory func() (*redis.Client, error)

// Deduplicator implements event deduplication using Redis.
type Deduplicator struct {
	Redis      *redis.Client
	Expiration time.Duration
	Mode       DedupeMode

	mu        sync.RWMutex
	newClient ClientFactory
}

func NewDeduplicator(redisClient *redis.Client, expiration time.Duration, mode DedupeMode, newClient ClientFactory) *Deduplicator {
	return &Deduplicator{
		Redis:      redisClient,
		Expiration: expiration,
		Mode:       mode,
		newClient:  newClient,
	}
}

func (d *Deduplicator) client() (*redis.Client, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.Redis == nil {
		return nil, errors.New("redis client not initialized")
	}

	return d.Redis, nil
}

// IsUnique checks if an event is unique AND adds it to the deduplication index.
func (d *Deduplicator) IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error) {
	client, err := d.client()
	if err != nil {
		return false, err
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
		keyHash := GetKeyHash(item.Key())
		return d.setKey(ctx, keyHash)
	case DedupeModeKeyHashMigration:
		keyHash := GetKeyHash(item.Key())
		isUnique, err := d.setKey(ctx, keyHash)
		if err != nil {
			return false, err
		}

		// Migration to the new hashing format
		if isUnique {
			// We might have succeeded setting the key only because the key just exist in the old format
			// Let's check if the old key exists
			isSet, err := client.Exists(ctx, item.Key()).Result()
			if err != nil {
				return false, err
			}

			keyExists := isSet == 1

			return !keyExists, nil
		}
	}

	return false, fmt.Errorf("unknown status")
}

func (d *Deduplicator) setKey(ctx context.Context, key string) (bool, error) {
	client, err := d.client()
	if err != nil {
		return false, err
	}

	status, err := client.SetArgs(ctx, key, "", redis.SetArgs{
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
func (d *Deduplicator) CheckUnique(ctx context.Context, item dedupe.Item) (bool, error) {
	client, err := d.client()
	if err != nil {
		return false, err
	}

	keysToCheck := make([]string, 0, 2)
	switch d.Mode {
	case DedupeModeRawKey:
		keysToCheck = append(keysToCheck, item.Key())
	case DedupeModeKeyHash:
		keysToCheck = append(keysToCheck, GetKeyHash(item.Key()))
	case DedupeModeKeyHashMigration:
		keysToCheck = append(keysToCheck, item.Key(), GetKeyHash(item.Key()))
	}

	isSet, err := client.Exists(ctx, keysToCheck...).Result()
	if err != nil {
		return false, err
	}

	return isSet == 0, nil
}

// Set sets events into redis
func (d *Deduplicator) Set(ctx context.Context, items ...dedupe.Item) ([]dedupe.Item, error) {
	client, err := d.client()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(items))
	for _, item := range items {
		switch d.Mode {
		case DedupeModeRawKey:
			keys = append(keys, item.Key())
		case DedupeModeKeyHash, DedupeModeKeyHashMigration:
			keys = append(keys, GetKeyHash(item.Key()))
		}
	}

	cmds, err := client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
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
		writeErr := fmt.Errorf("failed to set multiple keys in redis: %w", err)
		if shouldReconnectAfterWrite(ctx, writeErr) && d.newClient != nil {
			// Primary failover can leave stale sockets in the go-redis pool.
			// Replacing the whole client forces the next sink retry to create fresh
			// connections and re-resolve the configured endpoint.
			if reconnectErr := d.reconnect(); reconnectErr != nil {
				return nil, errors.Join(writeErr, fmt.Errorf("failed to reconnect redis client: %w", reconnectErr))
			}
		}

		return nil, writeErr
	}

	// Let's check if all the keys were created or if some of them already existed
	existingItems := []dedupe.Item{}
	for i, cmd := range cmds {
		item := items[i]
		if cmd.Err() != nil {
			if !errors.Is(cmd.Err(), redis.Nil) {
				writeErr := fmt.Errorf("failed to set key %s in redis: %w", item.Key(), cmd.Err())
				if shouldReconnectAfterWrite(ctx, writeErr) && d.newClient != nil {
					// Primary failover can leave stale sockets in the go-redis pool.
					// Replacing the whole client forces the next sink retry to create fresh
					// connections and re-resolve the configured endpoint.
					if reconnectErr := d.reconnect(); reconnectErr != nil {
						return nil, errors.Join(writeErr, fmt.Errorf("failed to reconnect redis client: %w", reconnectErr))
					}
				}

				return nil, writeErr
			}

			existingItems = append(existingItems, item)
		}
	}

	return existingItems, nil
}

func shouldReconnectAfterWrite(ctx context.Context, err error) bool {
	// nil and redis.Nil are normal outcomes for SET NX: redis.Nil means the key
	// already existed, not that the connection or primary endpoint is unhealthy.
	if err == nil || errors.Is(err, redis.Nil) {
		return false
	}

	// Once the caller context is done, the retry loop is already bounded by
	// cancellation. Reconnecting would create a fresh pool for an operation that
	// is no longer allowed to continue.
	if ctx.Err() != nil {
		return false
	}

	// Caller cancellation is not a Redis failover signal. It should stop the
	// current operation without forcing unrelated future operations onto a new pool.
	if errors.Is(err, context.Canceled) {
		return false
	}

	// A bare context deadline is not enough to prove the Redis pool is stale. If
	// the failure is a socket timeout, the net.Error checks below catch it.
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Transport failures mean the checked-out socket is gone or unusable. During
	// primary failover this is the common stale-pool failure mode.
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
		return true
	}

	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.EPIPE) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// READONLY means the endpoint we reached is not writable. That is the Redis
	// role signal we expect when a stale connection survives primary failover.
	return redis.IsReadOnlyError(err) || redis.HasErrorPrefix(err, "READONLY")
}

func (d *Deduplicator) reconnect() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	newClient, err := d.newClient()
	if err != nil {
		return err
	}

	oldClient := d.Redis
	d.Redis = newClient

	if oldClient != nil && oldClient != newClient {
		return oldClient.Close()
	}

	return nil
}

// Close closes underlying redis client
func (d *Deduplicator) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.Redis != nil {
		return d.Redis.Close()
	}

	return nil
}

var ErrNoDedupItems = errors.New("no dedup items provided")

func (d *Deduplicator) CheckUniqueBatch(ctx context.Context, items []dedupe.Item) (dedupe.CheckUniqueBatchResult, error) {
	client, err := d.client()
	if err != nil {
		return dedupe.CheckUniqueBatchResult{}, err
	}

	if len(items) == 0 {
		return dedupe.CheckUniqueBatchResult{}, ErrNoDedupItems
	}

	keys := make([]string, 0, len(items))
	for _, item := range items {
		switch d.Mode {
		case DedupeModeRawKey:
			keys = append(keys, item.Key())
		case DedupeModeKeyHash, DedupeModeKeyHashMigration:
			keys = append(keys, GetKeyHash(item.Key()))
		}
	}

	cmdResults, err := client.MGet(ctx, keys...).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return dedupe.CheckUniqueBatchResult{}, fmt.Errorf("failed to get multiple keys in redis: %w", err)
	}

	if len(cmdResults) != len(items) {
		return dedupe.CheckUniqueBatchResult{}, fmt.Errorf("failed to get all keys in redis")
	}

	result := dedupe.CheckUniqueBatchResult{
		UniqueItems:           make(dedupe.ItemSet, len(items)),
		AlreadyProcessedItems: make(dedupe.ItemSet, len(items)),
	}

	for i, cmdResult := range cmdResults {
		if cmdResult != nil {
			result.AlreadyProcessedItems[items[i]] = struct{}{}
			continue
		}

		result.UniqueItems[items[i]] = struct{}{}
	}

	return result, nil
}

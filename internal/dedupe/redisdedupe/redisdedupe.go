// Package redisdedupe implements event deduplication using Redis.
package redisdedupe

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/redis/go-redis/v9"

	"github.com/openmeterio/openmeter/internal/dedupe"
)

// Deduplicator implements event deduplication using Redis.
type Deduplicator struct {
	Redis      *redis.Client
	Expiration time.Duration
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

	status, err := d.Redis.SetArgs(ctx, item.Key(), "", redis.SetArgs{
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
	isSet, err := d.Redis.Exists(ctx, item.Key()).Result()
	if err != nil {
		return false, err
	}
	return isSet == 0, nil
}

// Set sets events into redis
func (d Deduplicator) Set(ctx context.Context, items ...dedupe.Item) error {
	for _, item := range items {
		// TODO: do it in batches if possible
		err := d.Redis.SetNX(ctx, item.Key(), "", d.Expiration).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

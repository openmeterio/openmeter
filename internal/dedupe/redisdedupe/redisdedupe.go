// Package redisdedupe implements event deduplication using Redis.
package redisdedupe

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/redis/go-redis/v9"
)

// Deduplicator implements event deduplication using Redis.
type Deduplicator struct {
	Redis      *redis.Client
	Expiration time.Duration
}

// IsUnique checks if the event is unique based on the key
func (d Deduplicator) IsUnique(ctx context.Context, item dedupe.Item) (bool, error) {
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

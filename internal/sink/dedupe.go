package sink

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/redis/go-redis/v9"
)

type Dedupe struct {
	config *DedupeConfig
}

func NewDedupe(config *DedupeConfig) *Dedupe {
	if config.Expiration == 0 {
		config.Expiration = 24 * time.Hour
	}

	return &Dedupe{
		config,
	}
}

type DedupeConfig struct {
	Redis      *redis.Client
	Expiration time.Duration
}

// IsUnique checks if the event is unique based on the key
func (d *Dedupe) IsUnique(ctx context.Context, event serializer.CloudEventsKafkaPayload) (bool, error) {
	isSet, err := d.config.Redis.Exists(ctx, event.GetKey()).Result()
	if err != nil {
		return false, err
	}
	return isSet == 0, nil
}

// Set sets events into redis
func (d *Dedupe) Set(ctx context.Context, events ...*serializer.CloudEventsKafkaPayload) error {
	for _, event := range events {
		// TODO: do it in batches if possible
		err := d.config.Redis.SetNX(ctx, event.GetKey(), "", d.config.Expiration).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

package redisdedupe

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/internal/ingest"
)

type Collector struct {
	config CollectorConfig
}

type CollectorConfig struct {
	Logger     *slog.Logger
	Redis      *redis.Client
	Expiration time.Duration
	Collector  ingest.Collector
}

func NewCollector(config CollectorConfig) (*Collector, error) {
	if config.Collector == nil {
		return nil, fmt.Errorf("collector is nil")
	}

	collector := &Collector{
		config: config,
	}
	return collector, nil
}

// TODO: pass context to Ingest
func (c Collector) Ingest(ev event.Event, namespace string) error {
	ctx := context.TODO()

	isUnique, err := c.isUnique(ctx, namespace, ev)
	if err != nil {
		return err
	}

	if isUnique {
		return c.config.Collector.Ingest(ev, namespace)
	}

	return nil
}

func (c Collector) Close() {
	c.config.Collector.Close()
}

// IsUnique checks if the entry is unique based on the key and sets it in store
func (c *Collector) isUnique(ctx context.Context, namespace string, ev event.Event) (bool, error) {
	status, err := c.config.Redis.SetArgs(ctx, ingest.GetEventKey(namespace, ev), "", redis.SetArgs{
		TTL:  c.config.Expiration,
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

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
	isSet, err := c.config.Redis.Exists(ctx, ingest.GetEventKey(namespace, ev)).Result()
	if err != nil {
		return false, err
	}
	return isSet == 0, nil
}

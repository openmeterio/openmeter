package memorydedupe

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/openmeterio/openmeter/internal/ingest"
)

type Collector struct {
	store     *lru.Cache[string, any]
	collector ingest.Collector
}

type CollectorConfig struct {
	Collector ingest.Collector
	Size      int
}

func NewCollector(config CollectorConfig) (*Collector, error) {
	if config.Size == 0 {
		return nil, fmt.Errorf("size cannot be 0")
	}
	if config.Collector == nil {
		return nil, fmt.Errorf("collector is nil")
	}

	store, err := lru.New[string, any](config.Size)
	if err != nil {
		return nil, err
	}

	dedupe := &Collector{
		store:     store,
		collector: config.Collector,
	}

	return dedupe, nil
}

// TODO: pass contect to Ingest
func (c Collector) Ingest(ev event.Event, namespace string) error {
	ctx := context.TODO()

	isUnique, err := c.IsUnique(ctx, namespace, ev)
	if err != nil {
		return err
	}

	if isUnique {
		return c.collector.Ingest(ev, namespace)
	}

	return nil
}

func (c Collector) Close() {
	c.collector.Close()
}

func (c Collector) IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error) {
	isContained, _ := c.store.ContainsOrAdd(dedupe.GetEventKey(namespace, ev), nil)
	return !isContained, nil
}

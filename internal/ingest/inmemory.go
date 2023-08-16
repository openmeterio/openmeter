package ingest

import (
	"sync"

	"github.com/cloudevents/sdk-go/v2/event"
	"golang.org/x/exp/maps"
)

// InMemoryCollector is a {Collector} backed by in-memory storage.
type InMemoryCollector struct {
	events map[string][]event.Event

	mu       sync.Mutex
	initOnce sync.Once
}

// NewInMemoryCollector returns a new {Collector} backed by in-memory storage.
func NewInMemoryCollector() *InMemoryCollector {
	return &InMemoryCollector{}
}

func (c *InMemoryCollector) init() {
	c.initOnce.Do(func() {
		c.events = make(map[string][]event.Event)
	})
}

// Ingest implements the {Collector} interface.
func (c *InMemoryCollector) Ingest(ev event.Event, namespace string) error {
	c.init()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.events[namespace] = append(c.events[namespace], ev)

	return nil
}

// Close implements the {Collector} interface.
func (c *InMemoryCollector) Close() {}

// Events returns events ingested into a namespace.
func (c *InMemoryCollector) Events(namespace string) []event.Event {
	c.init()

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.events[namespace]
}

// Namespaces returns namespaces events were ingested into.
func (c *InMemoryCollector) Namespaces() []string {
	c.init()

	c.mu.Lock()
	defer c.mu.Unlock()

	return maps.Keys(c.events)
}

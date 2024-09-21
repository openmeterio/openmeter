// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ingest

import (
	"context"
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
func (c *InMemoryCollector) Ingest(_ context.Context, namespace string, ev event.Event) error {
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

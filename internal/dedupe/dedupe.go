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

// Package dedupe implements in-process event deduplication.
package dedupe

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
)

// Deduplicator checks if an event is unique.
type Deduplicator interface {
	// IsUnique checks if an event is unique AND adds it to the deduplication index.
	// TODO: deprecate or rename IsUnique
	IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error)
	// CheckUnique checks if an item is unique.
	CheckUnique(ctx context.Context, item Item) (bool, error)
	// Set adds the item(s) to the deduplicator
	Set(ctx context.Context, events ...Item) error
	// Close cleans up resources
	Close() error
}

type Item struct {
	Namespace string
	ID        string
	Source    string
}

func (i Item) Key() string {
	return fmt.Sprintf("%s-%s-%s", i.Namespace, i.Source, i.ID)
}

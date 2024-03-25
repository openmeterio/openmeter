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

// Package ingest implements event ingestion.
package ingest

import (
	"context"

	"github.com/cloudevents/sdk-go/v2/event"
)

// Collector is a receiver of events that handles sending those events to some downstream broker.
type Collector interface {
	Ingest(ctx context.Context, namespace string, ev event.Event) error
	Close()
}

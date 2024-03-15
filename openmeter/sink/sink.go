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

// Package that implements event sink logic.
package sink

import (
	"github.com/openmeterio/openmeter/internal/sink"
)

// Sink is a streaming sink processor.
type (
	Sink                    = sink.Sink
	SinkConfig              = sink.SinkConfig
	Storage                 = sink.Storage
	ClickHouseStorage       = sink.ClickHouseStorage
	ClickHouseStorageConfig = sink.ClickHouseStorageConfig
)

// NewSink returns a sink processor.
func NewSink(config SinkConfig) (*sink.Sink, error) {
	return sink.NewSink(config)
}

// NewClickhouseStorage returns a ClickHouse Storage.
func NewClickhouseStorage(config ClickHouseStorageConfig) *sink.ClickHouseStorage {
	return sink.NewClickhouseStorage(config)
}

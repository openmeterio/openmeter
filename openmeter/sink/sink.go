// Package that implements event sink logic.
package sink

import (
	"github.com/openmeterio/openmeter/internal/sink"
)

// Sink is a streaming sink processor.
type Sink = sink.Sink
type SinkConfig = sink.SinkConfig
type Storage = sink.Storage
type SinkMessage = sink.SinkMessage
type ClickHouseStorage = sink.ClickHouseStorage
type ClickHouseStorageConfig = sink.ClickHouseStorageConfig

// NewSink returns a sink processor.
func NewSink(config SinkConfig) (*sink.Sink, error) {
	return sink.NewSink(config)
}

// NewClickhouseStorage returns a ClickHouse Storage.
func NewClickhouseStorage(config ClickHouseStorageConfig) *sink.ClickHouseStorage {
	return sink.NewClickhouseStorage(config)
}

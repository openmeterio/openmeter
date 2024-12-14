package common

import (
	"github.com/google/wire"
)

var Ingest = wire.NewSet(
	NewKafkaIngestCollector,
	NewIngestCollector,
)

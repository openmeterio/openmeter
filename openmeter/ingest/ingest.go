// Package ingest implements event ingestion.
package ingest

import (
	"github.com/openmeterio/openmeter/internal/ingest"
)

// Collector is a receiver of events that handles sending those events to some downstream broker.
type Collector = ingest.Collector

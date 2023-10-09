package ingest

import (
	"github.com/openmeterio/openmeter/internal/ingest"
)

// Deduplicator checks if an event is unique.
type Deduplicator = ingest.Deduplicator

// DeduplicatingCollector implements event deduplication at event ingestion.
type DeduplicatingCollector = ingest.DeduplicatingCollector

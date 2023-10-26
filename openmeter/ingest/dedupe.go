package ingest

import (
	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/openmeterio/openmeter/internal/ingest"
)

// Deduplicator checks if an event is unique.
type Deduplicator = dedupe.Deduplicator

// DeduplicatingCollector implements event deduplication at event ingestion.
type DeduplicatingCollector = ingest.DeduplicatingCollector

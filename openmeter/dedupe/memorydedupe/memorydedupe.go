// Package memorydedupe implements in-memory event deduplication.
package memorydedupe

import (
	"github.com/openmeterio/openmeter/internal/dedupe/memorydedupe"
)

// Deduplicator implements in-memory event deduplication.
type Deduplicator = memorydedupe.Deduplicator

// NewDeduplicator returns a new {Deduplicator}.
func NewDeduplicator(size int) (*Deduplicator, error) {
	return memorydedupe.NewDeduplicator(size)
}

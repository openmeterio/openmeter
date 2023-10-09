// Package redisdedupe implements event deduplication using Redis.
package redisdedupe

import (
	"github.com/openmeterio/openmeter/internal/dedupe/redisdedupe"
)

// Deduplicator implements event deduplication using Redis.
type Deduplicator = redisdedupe.Deduplicator

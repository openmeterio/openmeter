// Package dedupe implements in-process event deduplication.
package dedupe

import (
	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/internal/dedupe"
)

// GetEventKey creates a unique key from an event.
func GetEventKey(namespace string, ev event.Event) string {
	return dedupe.GetEventKey(namespace, ev)
}

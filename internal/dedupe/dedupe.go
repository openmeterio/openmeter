// Package dedupe implements in-process event deduplication.
package dedupe

import (
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
)

// GetEventKey creates a unique key from an event.
func GetEventKey(namespace string, ev event.Event) string {
	return fmt.Sprintf("%s-%s-%s", namespace, ev.Source(), ev.ID())
}

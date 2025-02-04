package billing

import (
	"fmt"
	"slices"
)

type AdvancementStrategy string

const (
	// ForgegroundAdvancementStrategy is the strategy where the invoice is advanced immediately as part
	// of the same transaction. Should be used in workers as advancement might take long time and we don't want
	// to block a HTTP request for that long.
	ForegroundAdvancementStrategy AdvancementStrategy = "foreground"
	// QueuedAdvancementStrategy is the strategy where the invoice is advanced in a separate worker (billing-worker).
	// This is useful for cases where the advancement might take a long time and we don't want to block the current
	// HTTP request.
	QueuedAdvancementStrategy AdvancementStrategy = "queued"
)

func (s AdvancementStrategy) Validate() error {
	if !slices.Contains(
		[]AdvancementStrategy{
			ForegroundAdvancementStrategy,
			QueuedAdvancementStrategy,
		},
		s,
	) {
		return fmt.Errorf("invalid advancement strategy: %s", s)
	}

	return nil
}

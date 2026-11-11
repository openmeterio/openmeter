package worker

import (
	"context"
	"errors"
	"time"
)

// The sink assigns row stored_at before writing, then timestamps this notification
// after the write succeeds. Credit rating uses stored_at < calculation Unix second,
// so even a persisted batch is invisible until the notification's second closes.
func waitForCreditUsage(ctx context.Context, storedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if storedAt.IsZero() {
		return errors.New("ingest batch storedAt is required for credit invalidation")
	}
	visibleAt := storedAt.Truncate(time.Second).Add(time.Second)
	delay := time.Until(visibleAt)
	if delay <= 0 {
		return nil
	}
	// Production handlers have a processing deadline and normal retry/DLQ policy.
	// A caller disabling that deadline must not pin a consumer on a future clock.
	if _, bounded := ctx.Deadline(); !bounded && delay > time.Second {
		return errors.New("ingest batch storedAt is ahead of the credit visibility window")
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

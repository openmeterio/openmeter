package grant

import (
	"math"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type (
	Owner           string
	NamespacedOwner struct {
		Namespace string
		ID        Owner
	}
)

// Casts the NamespacedGrantOwner to a NamespacedID. Owner might not be a valid ID.
func (n NamespacedOwner) NamespacedID() models.NamespacedID {
	return models.NamespacedID{
		Namespace: n.Namespace,
		ID:        string(n.ID),
	}
}

// Grant is an immutable definition used to increase balance.
type Grant struct {
	models.ManagedModel
	models.NamespacedModel

	// ID is the readonly identifies of a grant.
	ID string `json:"id,omitempty"`

	// Generic Owner reference
	OwnerID Owner `json:"owner"`

	// Amount The amount to grant. Can be positive or negative number.
	Amount float64 `json:"amount"`

	// Priority is a positive decimal numbers. With lower numbers indicating higher importance;
	// for example, a priority of 1 is more urgent than a priority of 2.
	// When there are several credit grants available for a single invoice, the system selects the credit with the highest priority.
	// In cases where credit grants share the same priority level, the grant closest to its expiration will be used first.
	// In the case of two credits have identical priorities and expiration dates, the system will use the credit that was created first.
	Priority uint8 `json:"priority"`

	// EffectiveAt The effective date.
	EffectiveAt time.Time `json:"effectiveAt"`

	// Expiration The expiration configuration.
	Expiration ExpirationPeriod `json:"expiration"`
	// ExpiresAt contains the exact expiration date calculated from effectiveAt and Expiration for rendering.
	// ExpiresAt is exclusive, meaning that the grant is no longer active after this time, but it is still active at the time.
	ExpiresAt time.Time `json:"expiresAt"`

	Metadata map[string]string `json:"metadata,omitempty"`

	// For user initiated voiding of the grant.
	VoidedAt *time.Time `json:"voidedAt,omitempty"`

	// How much of the grant can be rolled over after a reset operation.
	// Balance after a reset will be between ResetMinRollover and ResetMaxRollover.
	ResetMaxRollover float64 `json:"resetMaxRollover"`

	// How much balance the grant must have after a reset.
	// Balance after a reset will be between ResetMinRollover and ResetMaxRollover.
	ResetMinRollover float64 `json:"resetMinRollover"`

	// Recurrence config for the grant. If nil the grant doesn't recur.
	Recurrence *recurrence.Recurrence `json:"recurrence,omitempty"`
}

// Calculates expiration from effectiveAt and Expiration.
func (g Grant) GetExpiration() time.Time {
	return g.Expiration.GetExpiration(g.EffectiveAt)
}

func (g Grant) ActiveAt(t time.Time) bool {
	if g.DeletedAt != nil {
		if g.DeletedAt.Before(t) || g.DeletedAt.Equal(t) {
			return false
		}
	}
	if g.VoidedAt != nil {
		if g.VoidedAt.Before(t) || g.VoidedAt.Equal(t) {
			return false
		}
	}
	return (g.EffectiveAt.Before(t) || g.EffectiveAt.Equal(t)) && g.ExpiresAt.After(t)
}

// Calculates the new balance after a recurrence from the current balance
func (g Grant) RecurrenceBalance(currentBalance float64) float64 {
	// if it was wrongfully called on a non-recurring grant do nothing
	if g.Recurrence == nil {
		return currentBalance
	}

	// We have no rollover settings for recurring grants
	return g.Amount
}

// Calculates the new balance after a rollover from the current balance
func (g Grant) RolloverBalance(currentBalance float64) float64 {
	// At a rollover the maximum balance that can remain is the ResetMaxRollover,
	// while the minimum that has to be granted is ResetMinRollover.
	return math.Min(g.ResetMaxRollover, math.Max(g.ResetMinRollover, currentBalance))
}

func (g Grant) GetNamespacedID() models.NamespacedID {
	return models.NamespacedID{
		Namespace: g.Namespace,
		ID:        g.ID,
	}
}

func (g Grant) GetNamespacedOwner() NamespacedOwner {
	return NamespacedOwner{
		Namespace: g.Namespace,
		ID:        g.OwnerID,
	}
}

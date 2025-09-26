package grant

import (
	"math"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Grant is an immutable definition used to increase balance.
type Grant struct {
	models.ManagedModel
	models.NamespacedModel

	// ID is the readonly identifies of a grant.
	ID string `json:"id,omitempty"`

	// Generic Owner reference
	OwnerID string `json:"owner"`

	// Amount The amount to grant. Must be positive.
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
	Expiration *ExpirationPeriod `json:"expiration,omitempty"`
	// ExpiresAt contains the exact expiration date calculated from effectiveAt and Expiration for rendering.
	// ExpiresAt is exclusive, meaning that the grant is no longer active after this time, but it is still active at the time.
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	Annotations models.Annotations `json:"annotations,omitempty"`

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
	Recurrence *timeutil.Recurrence `json:"recurrence,omitempty"`
}

func (g Grant) Validate() error {
	// TODO: there are no clear validation requirements now but lets implement the interface
	return nil
}

// Calculates expiration from effectiveAt and Expiration.
func (g Grant) GetExpiration() *time.Time {
	if g.Expiration == nil {
		return nil
	}

	return lo.ToPtr(g.Expiration.GetExpiration(g.EffectiveAt))
}

func (g Grant) GetEffectivePeriod() timeutil.StartBoundedPeriod {
	p := timeutil.StartBoundedPeriod{
		From: g.EffectiveAt,
		To:   g.ExpiresAt,
	}

	// Let's bound by deletion time
	if g.DeletedAt != nil {
		switch {
		case p.To == nil:
			p.To = g.DeletedAt
		case g.DeletedAt.Before(*p.To):
			p.To = g.DeletedAt
		}
	}

	if g.VoidedAt != nil {
		switch {
		case p.To == nil:
			p.To = g.VoidedAt
		case g.VoidedAt.Before(*p.To):
			p.To = g.VoidedAt
		}
	}

	if p.To != nil && p.To.Before(p.From) {
		p.To = &p.From
	}

	return p
}

func (g Grant) ActiveAt(t time.Time) bool {
	return g.GetEffectivePeriod().Contains(t)
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
func (g Grant) RolloverBalance(endingBalance float64) float64 {
	// At a rollover the maximum balance that can remain is the ResetMaxRollover,
	// while the minimum that has to be granted is ResetMinRollover.
	return math.Min(g.ResetMaxRollover, math.Max(g.ResetMinRollover, endingBalance))
}

func (g Grant) GetNamespacedID() models.NamespacedID {
	return models.NamespacedID{
		Namespace: g.Namespace,
		ID:        g.ID,
	}
}

func (g Grant) GetNamespacedOwner() models.NamespacedID {
	return models.NamespacedID{
		Namespace: g.Namespace,
		ID:        g.OwnerID,
	}
}

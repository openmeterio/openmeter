package credit

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type GrantID string
type NamespacedGrantID struct {
	Namespace string
	ID        GrantID
}

type GrantOwner string
type NamespacedGrantOwner struct {
	Namespace string
	ID        GrantOwner
}

func NewNamespacedGrantID(namespace string, id GrantID) NamespacedGrantID {
	return NamespacedGrantID{
		Namespace: namespace,
		ID:        id,
	}
}

// Grant is an immutable definition used to increase balance.
type Grant struct {
	models.ManagedModel
	models.NamespacedModel

	// ID is the readonly identifies of a grant.
	ID GrantID `json:"id,omitempty"`

	// Generic Owner reference
	OwnerID GrantOwner `json:"owner"`

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
	// ExpiresAt contains the exact expiration date calculated from effectiveAt and Expiration for rendering
	ExpiresAt time.Time `json:"expiresAt"`

	Metadata map[string]string `json:"metadata,omitempty"`

	// For user initiated voiding of the grant.
	VoidedAt *time.Time `json:"voidedAt,omitempty"`

	// How much of the grant can be rolled over after a reset operation.
	ResetMaxRollover float64 `json:"resetMaxRollover"`

	// Recurrence config for the grant. If nil the grant doesn't recur.
	Recurrence *Recurrence `json:"recurrence,omitempty"`
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

func (g Grant) GetNamespacedID() NamespacedGrantID {
	return NamespacedGrantID{
		Namespace: g.Namespace,
		ID:        g.ID,
	}
}

func (g Grant) GetNamespacedOwner() NamespacedGrantOwner {
	return NamespacedGrantOwner{
		Namespace: g.Namespace,
		ID:        g.OwnerID,
	}
}

// // Render implements the chi renderer interface.
// func (c Grant) Render(w http.ResponseWriter, r *http.Request) error {
// 	return nil
// }

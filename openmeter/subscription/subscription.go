package subscription

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type PlanRef struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

type CreateSubscriptionInput struct {
	Plan PlanRef

	CustomerId string `json:"customerId,omitempty"`
	Currency   models.CurrencyCode
	models.CadencedModel
}

type Subscription struct {
	models.NamespacedModel
	models.ManagedModel
	CreateSubscriptionInput

	ID string `json:"id,omitempty"`
}

// THIS MIGHT BE A REALLY BAD IDEA
// DUE TO SCHEMA MISMATCHES AS TIME PASSES......
type SubscriptionPatch struct {
	models.NamespacedModel
	models.ManagedModel

	ID             string `json:"id,omitempty"`
	SubscriptionId string `json:"subscriptionId,omitempty"`

	// Primary ordering happens via activation time
	ActiveFrom time.Time `json:"activeFrom,omitempty"`
	// Secondary ordering can be used as a tie-breaker
	SecondaryOrdering int `json:"secondaryOrdering,omitempty"`

	// Patch info
	Operation string          `json:"operation,omitempty"`
	Path      string          `json:"path,omitempty"`
	Value     json.RawMessage `json:"value,omitempty"`
}

func (s *SubscriptionPatch) AsPatch() (any, error) {
	// TODO: Version validation!

	p := &AnyPatch{
		Op:    s.Operation,
		Path:  s.Path,
		Value: s.Value,
	}

	ser, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to pre-serialize patch: %w", err)
	}

	return Deserialize(ser)
}

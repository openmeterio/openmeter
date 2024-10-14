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

type SubscriptionPatch struct {
	models.NamespacedModel
	models.ManagedModel

	ID             string `json:"id,omitempty"`
	SubscriptionId string `json:"subscriptionId,omitempty"`

	// Primary ordering happens via when the patch was applied
	AppliedAt time.Time `json:"appliedAt,omitempty"`
	// BatchIndex can be used as a tie-breaker secondary ordering
	BatchIndex int `json:"batchIndex,omitempty"`

	// Patch info
	Operation string `json:"operation,omitempty"`
	Path      string `json:"path,omitempty"`
	Value     any    `json:"value,omitempty"`
}

func (s *SubscriptionPatch) AsPatch() (any, error) {
	p := &AnyPatch{
		Op:    s.Operation,
		Path:  s.Path,
		Value: s.Value,
	}

	// FIXME: This is a bit of a hack, parsing and deserialization are too coupled

	ser, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to pre-serialize patch: %w", err)
	}

	return Deserialize(ser)
}

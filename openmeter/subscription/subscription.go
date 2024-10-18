package subscription

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PlanRef struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

type SubscriptionItemRef struct {
	SubscriptionId string `json:"subscriptionId"`
	PhaseKey       string `json:"phaseKey"`
	ItemKey        string `json:"itemKey"`
}

func (r SubscriptionItemRef) Equals(r2 SubscriptionItemRef) bool {
	if r.SubscriptionId != r2.SubscriptionId {
		return false
	}
	if r.PhaseKey != r2.PhaseKey {
		return false
	}
	if r.ItemKey != r2.ItemKey {
		return false
	}
	return true
}

type CreateSubscriptionInput struct {
	Plan PlanRef

	CustomerId string `json:"customerId,omitempty"`
	Currency   currencyx.Code
	models.CadencedModel
}

type Subscription struct {
	models.NamespacedModel
	models.ManagedModel
	CreateSubscriptionInput

	ID string `json:"id,omitempty"`
}

type CreateSubscriptionPatchInput struct {
	AppliedAt  time.Time `json:"appliedAt,omitempty"`
	BatchIndex int       `json:"batchIndex,omitempty"`

	Patch
}

func TransformPatchesForRepository(patches []Patch, appliedAt time.Time) ([]CreateSubscriptionPatchInput, error) {
	var res []CreateSubscriptionPatchInput

	for i, p := range patches {
		pi := CreateSubscriptionPatchInput{
			Patch: p,
		}

		pi.AppliedAt = appliedAt
		pi.BatchIndex = i

		res = append(res, pi)
	}

	return res, nil
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
	p := &wPatch{
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

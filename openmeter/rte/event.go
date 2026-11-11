package rte

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CalculationKind string

const (
	CalculationKindCreditBalance      CalculationKind = "credit_balance"
	CalculationKindMeteredEntitlement CalculationKind = "metered_entitlement"
)

// CalculationTarget carries a concrete key and derives routing metadata from it.
type CalculationTarget struct {
	Kind CalculationKind `json:"kind"`
	Key  json.RawMessage `json:"key"`
}

type CalculationRequestedEvent struct {
	CalculationTarget
	Freshness *FreshnessPolicy `json:"freshness,omitempty"`
}

// FreshnessPolicy permits reuse of a clean result. TrackForSeconds controls
// how long source changes continue to mark the key dirty between pulls.
type FreshnessPolicy struct {
	TrackForSeconds int32 `json:"track_for_seconds"`
	MaxAgeSeconds   int32 `json:"max_age_seconds"`
}

func (e CalculationRequestedEvent) Validate() error {
	var err error
	if p := e.Freshness; p != nil && (p.TrackForSeconds <= 0 || p.MaxAgeSeconds < 0) {
		err = errors.New("track_for_seconds must be positive and max_age_seconds must be non-negative")
	}

	return models.NewNillableGenericValidationError(errors.Join(err, e.CalculationTarget.Validate()))
}

func NewCalculationRequestedEvent(key Key) (CalculationRequestedEvent, error) {
	target, err := NewCalculationTarget(key)
	return CalculationRequestedEvent{CalculationTarget: target}, err
}

func NewCalculationTarget(key Key) (CalculationTarget, error) {
	if err := key.Validate(); err != nil {
		return CalculationTarget{}, err
	}

	var kind CalculationKind
	switch key.(type) {
	case creditbalance.Key:
		kind = CalculationKindCreditBalance
	case meteredentitlement.Key:
		kind = CalculationKindMeteredEntitlement
	default:
		return CalculationTarget{}, models.NewGenericValidationError(fmt.Errorf("unsupported calculation key %T", key))
	}

	encoded, err := json.Marshal(key)
	if err != nil {
		return CalculationTarget{}, fmt.Errorf("encode calculation key: %w", err)
	}

	return CalculationTarget{Kind: kind, Key: encoded}, nil
}

func (e CalculationRequestedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{Subsystem: "rte", Name: "calculationRequested", Version: "v1"})
}

// DecodeKey validates the kind and concrete identity together at the event boundary.
func (e CalculationTarget) DecodeKey() (Key, error) {
	var key Key
	switch e.Kind {
	case CalculationKindCreditBalance:
		var decoded creditbalance.Key
		if err := json.Unmarshal(e.Key, &decoded); err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("credit balance key: %w", err))
		}

		key = decoded
	case CalculationKindMeteredEntitlement:
		var decoded meteredentitlement.Key
		if err := json.Unmarshal(e.Key, &decoded); err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("metered entitlement key: %w", err))
		}

		key = decoded
	default:
		return nil, models.NewGenericValidationError(fmt.Errorf("unsupported calculation kind %q", e.Kind))
	}

	if err := key.Validate(); err != nil {
		return nil, err
	}

	return key, nil
}

func (e CalculationTarget) Validate() error {
	_, err := e.DecodeKey()
	return err
}

func (e CalculationTarget) EventMetadata() metadata.EventMetadata {
	key, err := e.DecodeKey()
	if err != nil {
		return metadata.EventMetadata{}
	}

	var namespace, subject string
	switch key := key.(type) {
	case creditbalance.Key:
		namespace = key.CustomerID.Namespace
		subject = metadata.ComposeResourcePath(namespace, metadata.EntityCustomer, key.CustomerID.ID)
	case meteredentitlement.Key:
		namespace = key.EntitlementID.Namespace
		subject = metadata.ComposeResourcePath(namespace, metadata.EntityEntitlement, key.EntitlementID.ID)
	}

	return metadata.EventMetadata{Source: metadata.ComposeResourcePath(namespace, "rte"), Subject: subject}
}

// CalculationUpdatedEvent describes the state accepted by persistence. Delivery
// may repeat or arrive out of order; CalculatedAt identifies the stored version.
type CalculationUpdatedEvent struct {
	CalculationTarget
	State json.RawMessage `json:"state"`
}

func NewCalculationUpdatedEvent[K Key, Result any](input StoreInput[K, Result]) (CalculationUpdatedEvent, error) {
	if err := input.Validate(); err != nil {
		return CalculationUpdatedEvent{}, err
	}

	target, err := NewCalculationTarget(input.Key)
	if err != nil {
		return CalculationUpdatedEvent{}, err
	}

	encoded, err := json.Marshal(input.State)
	if err != nil {
		return CalculationUpdatedEvent{}, fmt.Errorf("encode calculation state: %w", err)
	}

	return CalculationUpdatedEvent{CalculationTarget: target, State: encoded}, nil
}

func (e CalculationUpdatedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{Subsystem: "rte", Name: "calculationUpdated", Version: "v1"})
}

func (e CalculationUpdatedEvent) Validate() error {
	if err := e.CalculationTarget.Validate(); err != nil {
		return err
	}

	var state State[json.RawMessage]
	if err := json.Unmarshal(e.State, &state); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("state: %w", err))
	}

	if state.CalculatedAt.IsZero() || len(state.Value) == 0 || string(state.Value) == "null" {
		return models.NewGenericValidationError(fmt.Errorf("state requires calculated_at and value"))
	}

	return nil
}

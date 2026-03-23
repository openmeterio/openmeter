package notification

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type RawPayload map[string]any

func AsRawPayload(t any) (RawPayload, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	var m RawPayload
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}

type EventPayloadMeta struct {
	Type    EventType `json:"type"`
	Version int       `json:"version,omitempty"`
}

func (m EventPayloadMeta) Validate() error {
	return m.Type.Validate()
}

const (
	// EventPayloadVersionLegacy is the implicit version for events stored before versioning was
	// introduced. Never explicitly written; absence of the field unmarshals to 0.
	EventPayloadVersionLegacy int = 0

	// EventPayloadVersionCurrent is the version written for all new events.
	EventPayloadVersionCurrent int = 1
)

// EventPayload is a union type capturing payload for all EventType of Events.
type EventPayload struct {
	EventPayloadMeta

	// Entitlements
	BalanceThreshold *BalanceThresholdPayload `json:"balanceThreshold,omitempty"`
	EntitlementReset *EntitlementResetPayload `json:"entitlementReset,omitempty"`

	// Invoice
	Invoice *InvoicePayload `json:"invoice,omitempty"`
}

func (p EventPayload) Validate() error {
	switch p.Type {
	case EventTypeBalanceThreshold:
		if p.BalanceThreshold == nil {
			return models.NewGenericValidationError(errors.New("missing balance threshold payload"))
		}

		return p.BalanceThreshold.Validate()
	case EventTypeEntitlementReset:
		if p.EntitlementReset == nil {
			return models.NewGenericValidationError(errors.New("missing entitlement reset payload"))
		}

		return p.EntitlementReset.Validate()
	case EventTypeInvoiceCreated, EventTypeInvoiceUpdated:
		if p.Invoice == nil {
			return models.NewGenericValidationError(errors.New("missing invoice payload"))
		}

		return p.Invoice.Validate()
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid event type: %s", p.Type))
	}
}

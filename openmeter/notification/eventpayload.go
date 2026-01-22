package notification

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
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
	Type EventType `json:"type"`
}

func (m EventPayloadMeta) Validate() error {
	return m.Type.Validate()
}

// EventPayload is a union type capturing payload for all EventType of Events.
type EventPayload struct {
	EventPayloadMeta

	// Entitlements
	BalanceThreshold *BalanceThresholdPayload `json:"balanceThreshold,omitempty"`
	EntitlementReset *EntitlementResetPayload `json:"entitlementReset,omitempty"`

	// Invoice
	Invoice *billing.EventStandardInvoice `json:"invoice,omitempty"`
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

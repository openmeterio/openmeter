package notification

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/api"
)

func PayloadToMapInterface(t any) (map[string]interface{}, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
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

	// Balance Threshold
	BalanceThreshold *BalanceThresholdPayload `json:"balanceThreshold,omitempty"`
}

func (p EventPayload) Validate() error {
	switch p.Type {
	case EventTypeBalanceThreshold:
		if p.BalanceThreshold == nil {
			return ValidationError{
				Err: errors.New("missing balance threshold payload"),
			}
		}

		return p.BalanceThreshold.Validate()
	default:
		return ValidationError{
			Err: fmt.Errorf("invalid event type: %s", p.Type),
		}
	}
}

type BalanceThresholdPayload struct {
	Entitlement api.EntitlementMetered                    `json:"entitlement"`
	Feature     api.Feature                               `json:"feature"`
	Subject     api.Subject                               `json:"subject"`
	Threshold   api.NotificationRuleBalanceThresholdValue `json:"threshold"`
	Value       api.EntitlementValue                      `json:"value"`
}

// Validate returns an error if balance threshold payload is invalid.
func (b BalanceThresholdPayload) Validate() error {
	return nil
}

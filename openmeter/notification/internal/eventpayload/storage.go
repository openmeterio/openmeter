package eventpayload

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/notification"
)

// FromDBRawMessage reconstructs the V1 payload from the immutable stored snapshot.
// Event identity and timestamp stay on the Event and are added by the wire mappers.
func FromDBRawMessage(data json.RawMessage, eventType notification.EventType) (notification.EventPayload, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return notification.EventPayload{}, fmt.Errorf("invalid notification event payload: %w", err)
	}

	required := []string{"entitlement", "feature", "subject", "value"}
	switch eventType {
	case notification.EventTypeBalanceThreshold:
		required = append(required, "threshold")
	case notification.EventTypeEntitlementReset:
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		required = []string{"id"}
	default:
		return notification.EventPayload{}, fmt.Errorf("invalid notification event type: %s", eventType)
	}

	for _, name := range required {
		if len(fields[name]) == 0 || string(fields[name]) == "null" {
			return notification.EventPayload{}, fmt.Errorf("invalid notification event payload: missing %s", name)
		}
	}

	mapped, err := mapJSONFields(data, toCamelCase)
	if err != nil {
		return notification.EventPayload{}, err
	}

	payload := notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{Type: eventType, Version: notification.EventPayloadVersionCurrent},
	}
	switch eventType {
	case notification.EventTypeBalanceThreshold:
		err = json.Unmarshal(mapped, &payload.BalanceThreshold)
	case notification.EventTypeEntitlementReset:
		err = json.Unmarshal(mapped, &payload.EntitlementReset)
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		err = json.Unmarshal(mapped, &payload.Invoice)
	}

	if err != nil {
		return notification.EventPayload{}, fmt.Errorf("invalid notification event payload: %w", err)
	}

	return payload, nil
}

// mapJSONFields changes model property names without changing numbers or user keys.
// Group-by filter keys are user-owned, but each filter's properties are model fields.
func mapJSONFields(data json.RawMessage, rename func(string) string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("invalid notification event payload: empty JSON")
	}

	switch trimmed[0] {
	case '{':
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return nil, err
		}

		mapped := make(map[string]json.RawMessage, len(fields))
		for name, value := range fields {
			var err error
			switch toCamelCase(name) {
			case "metadata", "annotations", "meterGroupByFilters":
			case "advancedMeterGroupByFilters":
				if string(value) != "null" {
					var filters map[string]json.RawMessage
					if err = json.Unmarshal(value, &filters); err != nil {
						return nil, err
					}

					for key, filter := range filters {
						filters[key], err = mapJSONFields(filter, rename)
						if err != nil {
							return nil, err
						}
					}

					value, err = json.Marshal(filters)
				}
			default:
				value, err = mapJSONFields(value, rename)
			}

			if err != nil {
				return nil, err
			}

			key := rename(name)
			if _, exists := mapped[key]; exists {
				return nil, fmt.Errorf("invalid notification event payload: duplicate mapped field %s", key)
			}

			mapped[key] = value
		}

		return json.Marshal(mapped)
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, err
		}

		for i, item := range items {
			mapped, err := mapJSONFields(item, rename)
			if err != nil {
				return nil, err
			}

			items[i] = mapped
		}

		return json.Marshal(items)
	default:
		if !json.Valid(data) {
			return nil, fmt.Errorf("invalid notification event payload JSON")
		}

		return data, nil
	}
}

func toCamelCase(name string) string {
	// Raw API union members retain their original JSON, so acronym spelling must
	// match V1 rather than relying on encoding/json's case-insensitive struct lookup.
	switch name {
	case "masked_api_key":
		return "maskedAPIKey"
	case "secret_api_key":
		return "secretAPIKey"
	case "interval_iso":
		return "intervalISO"
	}

	parts := strings.Split(name, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}

	return strings.Join(parts, "")
}

package eventpayload

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/notification"
)

// ToCanonicalJSON stores only the type-specific snapshot. Envelope metadata comes
// from the Event row when serving V1 or delivering its webhook representation.
func ToCanonicalJSON(payload notification.EventPayload) (json.RawMessage, error) {
	if err := payload.Validate(); err != nil {
		return nil, fmt.Errorf("invalid notification event payload: %w", err)
	}

	var data any
	switch payload.Type {
	case notification.EventTypeBalanceThreshold:
		data = payload.BalanceThreshold
	case notification.EventTypeEntitlementReset:
		data = payload.EntitlementReset
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		data = payload.Invoice
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize notification event payload: %w", err)
	}

	return mapJSONFields(raw, toSnakeCase)
}

var (
	snakeCaseAcronymBoundary = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
	snakeCaseWordBoundary    = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

// Keep this mapping aligned with the V3 legacy payloads and generated contracts.
func toSnakeCase(name string) string {
	name = snakeCaseAcronymBoundary.ReplaceAllString(name, "${1}_${2}")
	name = snakeCaseWordBoundary.ReplaceAllString(name, "${1}_${2}")

	return strings.ToLower(name)
}

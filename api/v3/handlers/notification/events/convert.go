package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/samber/lo"

	v1api "github.com/openmeterio/openmeter/api"
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
	"github.com/openmeterio/openmeter/pkg/models"
)

// The allow-list matches the orderings the adapter can apply; anything else would
// silently fall through to order-by-id.
func FromAPIEventSortField(ctx context.Context, field string) (notification.OrderBy, error) {
	switch field {
	case "id":
		return notification.OrderByID, nil
	case "type":
		return notification.OrderByType, nil
	case "created_at":
		return notification.OrderByCreatedAt, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(ctx, field, "id", "type", "created_at")
	}
}

// The v3 wire value is lowercase ("failed") while the column keeps the uppercase value
// written by v1 ("FAILED").
func FromAPIBillingNotificationEventDeliveryState(v api.BillingNotificationEventDeliveryState) (notification.EventDeliveryStatusState, error) {
	switch v {
	case api.BillingNotificationEventDeliveryStateSuccess:
		return notification.EventDeliveryStatusStateSuccess, nil
	case api.BillingNotificationEventDeliveryStateFailed:
		return notification.EventDeliveryStatusStateFailed, nil
	case api.BillingNotificationEventDeliveryStateSending:
		return notification.EventDeliveryStatusStateSending, nil
	case api.BillingNotificationEventDeliveryStatePending:
		return notification.EventDeliveryStatusStatePending, nil
	case api.BillingNotificationEventDeliveryStateResending:
		return notification.EventDeliveryStatusStateResending, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid notification event delivery state: %s", v))
	}
}

func ToAPIBillingNotificationEventDeliveryState(v notification.EventDeliveryStatusState) (api.BillingNotificationEventDeliveryState, error) {
	switch v {
	case notification.EventDeliveryStatusStateSuccess:
		return api.BillingNotificationEventDeliveryStateSuccess, nil
	case notification.EventDeliveryStatusStateFailed:
		return api.BillingNotificationEventDeliveryStateFailed, nil
	case notification.EventDeliveryStatusStateSending:
		return api.BillingNotificationEventDeliveryStateSending, nil
	case notification.EventDeliveryStatusStatePending:
		return api.BillingNotificationEventDeliveryStatePending, nil
	case notification.EventDeliveryStatusStateResending:
		return api.BillingNotificationEventDeliveryStateResending, nil
	default:
		return "", fmt.Errorf("invalid notification event delivery state: %s", v)
	}
}

// The stored value is the v1 API model, where `NUMBER` and `PERCENT` are deprecated
// aliases of `usage_value` and `usage_percentage`; events written before the rename
// must not leak the legacy spelling.
func ToAPIBillingNotificationBalanceThresholdType(v v1api.NotificationRuleBalanceThresholdValueType) (api.BillingNotificationBalanceThresholdType, error) {
	switch v {
	case v1api.NotificationRuleBalanceThresholdValueTypeBalanceValue:
		return api.BillingNotificationBalanceThresholdTypeBalanceValue, nil
	case v1api.NotificationRuleBalanceThresholdValueTypeUsagePercentage,
		v1api.NotificationRuleBalanceThresholdValueTypePercent:
		return api.BillingNotificationBalanceThresholdTypeUsagePercentage, nil
	case v1api.NotificationRuleBalanceThresholdValueTypeUsageValue,
		v1api.NotificationRuleBalanceThresholdValueTypeNumber:
		return api.BillingNotificationBalanceThresholdTypeUsageValue, nil
	default:
		return "", fmt.Errorf("invalid notification balance threshold type: %s", v)
	}
}

func ToAPIBillingNotificationEvent(e notification.Event) (api.BillingNotificationEvent, error) {
	deliveryStatus, err := ToAPIBillingNotificationEventDeliveryStatuses(e.DeliveryStatus)
	if err != nil {
		return api.BillingNotificationEvent{}, err
	}

	event := api.BillingNotificationEvent{
		Id:        e.ID,
		Type:      api.BillingNotificationEventType(e.Type),
		CreatedAt: e.CreatedAt,
		Rule: api.BillingNotificationRuleReference{
			Id:   e.Rule.ID,
			Type: api.BillingNotificationEventType(e.Rule.Type),
			Name: e.Rule.Name,
		},
		DeliveryStatus: deliveryStatus,
	}

	if err := setAPIEventPayload(&event, e); err != nil {
		return api.BillingNotificationEvent{}, err
	}

	return event, nil
}

// Unlike v1, the channel is reported by id only, so a channel that has since been
// disabled or deleted is still reported.
func ToAPIBillingNotificationEventDeliveryStatuses(statuses []notification.EventDeliveryStatus) ([]api.BillingNotificationEventDeliveryStatus, error) {
	result := make([]api.BillingNotificationEventDeliveryStatus, 0, len(statuses))

	for _, status := range statuses {
		state, err := ToAPIBillingNotificationEventDeliveryState(status.State)
		if err != nil {
			return nil, err
		}

		attempts, err := ToAPIBillingNotificationEventDeliveryAttempts(status.Attempts)
		if err != nil {
			return nil, err
		}

		result = append(result, api.BillingNotificationEventDeliveryStatus{
			ChannelId:   status.ChannelID,
			State:       state,
			Reason:      status.Reason,
			UpdatedAt:   status.UpdatedAt,
			NextAttempt: status.NextAttempt,
			Attempts:    attempts,
		})
	}

	return result, nil
}

func ToAPIBillingNotificationEventDeliveryAttempts(attempts []notification.EventDeliveryAttempt) ([]api.BillingNotificationEventDeliveryAttempt, error) {
	notification.SortEventDeliveryAttemptsInDescOrder(attempts)

	result := make([]api.BillingNotificationEventDeliveryAttempt, 0, len(attempts))

	for _, attempt := range attempts {
		state, err := ToAPIBillingNotificationEventDeliveryState(attempt.State)
		if err != nil {
			return nil, err
		}

		var statusCode *int32
		if attempt.Response.StatusCode != nil {
			statusCode = lo.ToPtr(int32(*attempt.Response.StatusCode))
		}

		result = append(result, api.BillingNotificationEventDeliveryAttempt{
			State:     state,
			Timestamp: attempt.Timestamp,
			Response: api.BillingNotificationEventDeliveryAttemptResponse{
				StatusCode: statusCode,
				Body:       attempt.Response.Body,
				DurationMs: attempt.Response.Duration.Milliseconds(),
				Url:        attempt.Response.URL,
			},
		})
	}

	return result, nil
}

// The v3 API returns the legacy payloads with the v1 content: build the v1 payload with
// the v1 mapping and rename its fields to snake_case, so the two APIs cannot drift.
func setAPIEventPayload(event *api.BillingNotificationEvent, e notification.Event) error {
	var (
		payload any
		err     error
	)

	switch e.Type {
	case notification.EventTypeBalanceThreshold:
		payload, err = httpdriver.FromEventAsBalanceThresholdPayload(e)
	case notification.EventTypeEntitlementReset:
		payload, err = httpdriver.FromEventAsEntitlementResetPayload(e)
	case notification.EventTypeInvoiceCreated:
		payload, err = httpdriver.FromEventAsInvoiceCreatedPayload(e)
	case notification.EventTypeInvoiceUpdated:
		payload, err = httpdriver.FromEventAsInvoiceUpdatedPayload(e)
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid notification event type: %s", e.Type))
	}

	if err != nil {
		return fmt.Errorf("failed to map payload of notification event %s: %w", e.ID, err)
	}

	v1Body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload of notification event %s: %w", e.ID, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(v1Body))
	// Keep numbers as written, float64 would round large or precise values.
	decoder.UseNumber()

	var v1Value any
	if err := decoder.Decode(&v1Value); err != nil {
		return fmt.Errorf("failed to decode payload of notification event %s: %w", e.ID, err)
	}

	body, err := json.Marshal(toSnakeCaseKeys(v1Value))
	if err != nil {
		return fmt.Errorf("failed to marshal payload of notification event %s: %w", e.ID, err)
	}

	return event.Payload.UnmarshalJSON(body)
}

// The keys of these v1 fields are user data, not field names, so they keep their case.
// The values of advancedMeterGroupByFilters are filter objects with v1 field names.
var (
	userKeyedFields = map[string]bool{
		"metadata":            true,
		"annotations":         true,
		"meterGroupByFilters": true,
	}
	userKeyedFieldsWithV1Values = map[string]bool{
		"advancedMeterGroupByFilters": true,
	}
)

func toSnakeCaseKeys(v any) any {
	switch v := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, value := range v {
			switch {
			case userKeyedFields[key]:
				out[toSnakeCase(key)] = value
			case userKeyedFieldsWithV1Values[key]:
				filters, ok := value.(map[string]any)
				if !ok {
					out[toSnakeCase(key)] = value

					continue
				}

				mapped := make(map[string]any, len(filters))
				for filterKey, filter := range filters {
					mapped[filterKey] = toSnakeCaseKeys(filter)
				}

				out[toSnakeCase(key)] = mapped
			default:
				out[toSnakeCase(key)] = toSnakeCaseKeys(value)
			}
		}

		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = toSnakeCaseKeys(item)
		}

		return out
	default:
		return v
	}
}

var (
	snakeCaseAcronymBoundary = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
	snakeCaseWordBoundary    = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

// toSnakeCase must match the renaming used to create the v3 copies of the v1 models,
// for example maskedAPIKey becomes masked_api_key.
func toSnakeCase(name string) string {
	name = snakeCaseAcronymBoundary.ReplaceAllString(name, "${1}_${2}")
	name = snakeCaseWordBoundary.ReplaceAllString(name, "${1}_${2}")

	return strings.ToLower(name)
}

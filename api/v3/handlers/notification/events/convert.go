package events

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	v1api "github.com/openmeterio/openmeter/api"
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/notification"
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
func ToDomainDeliveryState(v api.NotificationEventDeliveryState) (notification.EventDeliveryStatusState, error) {
	switch v {
	case api.NotificationEventDeliveryStateSuccess:
		return notification.EventDeliveryStatusStateSuccess, nil
	case api.NotificationEventDeliveryStateFailed:
		return notification.EventDeliveryStatusStateFailed, nil
	case api.NotificationEventDeliveryStateSending:
		return notification.EventDeliveryStatusStateSending, nil
	case api.NotificationEventDeliveryStatePending:
		return notification.EventDeliveryStatusStatePending, nil
	case api.NotificationEventDeliveryStateResending:
		return notification.EventDeliveryStatusStateResending, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid notification event delivery state: %s", v))
	}
}

func ToAPIDeliveryState(v notification.EventDeliveryStatusState) (api.NotificationEventDeliveryState, error) {
	switch v {
	case notification.EventDeliveryStatusStateSuccess:
		return api.NotificationEventDeliveryStateSuccess, nil
	case notification.EventDeliveryStatusStateFailed:
		return api.NotificationEventDeliveryStateFailed, nil
	case notification.EventDeliveryStatusStateSending:
		return api.NotificationEventDeliveryStateSending, nil
	case notification.EventDeliveryStatusStatePending:
		return api.NotificationEventDeliveryStatePending, nil
	case notification.EventDeliveryStatusStateResending:
		return api.NotificationEventDeliveryStateResending, nil
	default:
		return "", fmt.Errorf("invalid notification event delivery state: %s", v)
	}
}

// Only the v3 surface carries the `v1.` prefix; the domain constants keep the
// unversioned spelling shared with v1 and the stored column.
func ToAPIEventType(v notification.EventType) (api.NotificationEventType, error) {
	switch v {
	case notification.EventTypeBalanceThreshold:
		return api.NotificationEventTypeV1EntitlementsBalanceThreshold, nil
	case notification.EventTypeEntitlementReset:
		return api.NotificationEventTypeV1EntitlementsReset, nil
	case notification.EventTypeInvoiceCreated:
		return api.NotificationEventTypeV1InvoiceCreated, nil
	case notification.EventTypeInvoiceUpdated:
		return api.NotificationEventTypeV1InvoiceUpdated, nil
	default:
		return "", fmt.Errorf("invalid notification event type: %s", v)
	}
}

func ToDomainEventType(v api.NotificationEventType) (notification.EventType, error) {
	switch v {
	case api.NotificationEventTypeV1EntitlementsBalanceThreshold:
		return notification.EventTypeBalanceThreshold, nil
	case api.NotificationEventTypeV1EntitlementsReset:
		return notification.EventTypeEntitlementReset, nil
	case api.NotificationEventTypeV1InvoiceCreated:
		return notification.EventTypeInvoiceCreated, nil
	case api.NotificationEventTypeV1InvoiceUpdated:
		return notification.EventTypeInvoiceUpdated, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid notification event type: %s", v))
	}
}

// The stored value is the v1 API model, where `NUMBER` and `PERCENT` are deprecated
// aliases of `usage_value` and `usage_percentage`; events written before the rename
// must not leak the legacy spelling.
func ToAPIBalanceThresholdType(v v1api.NotificationRuleBalanceThresholdValueType) (api.NotificationEventBalanceThresholdType, error) {
	switch v {
	case v1api.NotificationRuleBalanceThresholdValueTypeBalanceValue:
		return api.NotificationEventBalanceThresholdTypeBalanceValue, nil
	case v1api.NotificationRuleBalanceThresholdValueTypeUsagePercentage,
		v1api.NotificationRuleBalanceThresholdValueTypePercent:
		return api.NotificationEventBalanceThresholdTypeUsagePercentage, nil
	case v1api.NotificationRuleBalanceThresholdValueTypeUsageValue,
		v1api.NotificationRuleBalanceThresholdValueTypeNumber:
		return api.NotificationEventBalanceThresholdTypeUsageValue, nil
	default:
		return "", fmt.Errorf("invalid notification balance threshold type: %s", v)
	}
}

func ToAPIEvent(e notification.Event) (api.NotificationEvent, error) {
	deliveryStatus, err := ToAPIDeliveryStatuses(e.DeliveryStatus)
	if err != nil {
		return api.NotificationEvent{}, err
	}

	eventType, err := ToAPIEventType(e.Type)
	if err != nil {
		return api.NotificationEvent{}, err
	}

	ruleType, err := ToAPIEventType(e.Rule.Type)
	if err != nil {
		return api.NotificationEvent{}, err
	}

	event := api.NotificationEvent{
		Id:        e.ID,
		Type:      eventType,
		CreatedAt: e.CreatedAt,
		Rule: api.NotificationRuleReference{
			Id:   e.Rule.ID,
			Type: ruleType,
			Name: e.Rule.Name,
		},
		DeliveryStatus: deliveryStatus,
	}

	if err := setAPIEventPayload(&event, e); err != nil {
		return api.NotificationEvent{}, err
	}

	return event, nil
}

// Unlike v1, the channel is reported by id only, so a channel that has since been
// disabled or deleted is still reported.
func ToAPIDeliveryStatuses(statuses []notification.EventDeliveryStatus) ([]api.NotificationEventDeliveryStatus, error) {
	result := make([]api.NotificationEventDeliveryStatus, 0, len(statuses))

	for _, status := range statuses {
		state, err := ToAPIDeliveryState(status.State)
		if err != nil {
			return nil, err
		}

		attempts, err := ToAPIDeliveryAttempts(status.Attempts)
		if err != nil {
			return nil, err
		}

		result = append(result, api.NotificationEventDeliveryStatus{
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

func ToAPIDeliveryAttempts(attempts []notification.EventDeliveryAttempt) ([]api.NotificationEventDeliveryAttempt, error) {
	notification.SortEventDeliveryAttemptsInDescOrder(attempts)

	result := make([]api.NotificationEventDeliveryAttempt, 0, len(attempts))

	for _, attempt := range attempts {
		state, err := ToAPIDeliveryState(attempt.State)
		if err != nil {
			return nil, err
		}

		var statusCode *int32
		if attempt.Response.StatusCode != nil {
			statusCode = lo.ToPtr(int32(*attempt.Response.StatusCode))
		}

		result = append(result, api.NotificationEventDeliveryAttempt{
			State:     state,
			Timestamp: attempt.Timestamp,
			Response: api.NotificationEventDeliveryAttemptResponse{
				StatusCode: statusCode,
				Body:       attempt.Response.Body,
				DurationMs: attempt.Response.Duration.Milliseconds(),
				Url:        attempt.Response.URL,
			},
		})
	}

	return result, nil
}

// The stored payload holds v1 API models; only the identifiers and scalars the v3
// surface exposes are extracted, never the v1 shapes.
func setAPIEventPayload(event *api.NotificationEvent, e notification.Event) error {
	switch e.Type {
	case notification.EventTypeBalanceThreshold:
		if e.Payload.BalanceThreshold == nil {
			return fmt.Errorf("missing balance threshold payload on notification event %s", e.ID)
		}

		thresholdType, err := ToAPIBalanceThresholdType(e.Payload.BalanceThreshold.Threshold.Type)
		if err != nil {
			return err
		}

		return event.Payload.FromNotificationEventBalanceThresholdPayload(api.NotificationEventBalanceThresholdPayload{
			Id:        e.ID,
			Type:      api.NotificationEventBalanceThresholdPayloadTypeV1EntitlementsBalanceThreshold,
			Timestamp: e.CreatedAt,
			Data: api.NotificationEventBalanceThresholdData{
				EntitlementId: e.Payload.BalanceThreshold.Entitlement.Id,
				Feature: api.NotificationEventFeatureReference{
					Id:  e.Payload.BalanceThreshold.Feature.Id,
					Key: e.Payload.BalanceThreshold.Feature.Key,
				},
				SubjectKey: e.Payload.BalanceThreshold.Subject.Key,
				CustomerId: lo.EmptyableToPtr(e.Payload.BalanceThreshold.Customer.Id),
				Value:      toAPIEntitlementValue(e.Payload.BalanceThreshold.Value),
				Threshold: api.NotificationEventBalanceThreshold{
					Type:  thresholdType,
					Value: e.Payload.BalanceThreshold.Threshold.Value,
				},
			},
		})

	case notification.EventTypeEntitlementReset:
		if e.Payload.EntitlementReset == nil {
			return fmt.Errorf("missing entitlement reset payload on notification event %s", e.ID)
		}

		return event.Payload.FromNotificationEventResetPayload(api.NotificationEventResetPayload{
			Id:        e.ID,
			Type:      api.NotificationEventResetPayloadTypeV1EntitlementsReset,
			Timestamp: e.CreatedAt,
			Data: api.NotificationEventEntitlementData{
				EntitlementId: e.Payload.EntitlementReset.Entitlement.Id,
				Feature: api.NotificationEventFeatureReference{
					Id:  e.Payload.EntitlementReset.Feature.Id,
					Key: e.Payload.EntitlementReset.Feature.Key,
				},
				SubjectKey: e.Payload.EntitlementReset.Subject.Key,
				CustomerId: lo.EmptyableToPtr(e.Payload.EntitlementReset.Customer.Id),
				Value:      toAPIEntitlementValue(e.Payload.EntitlementReset.Value),
			},
		})

	case notification.EventTypeInvoiceCreated:
		if e.Payload.Invoice == nil {
			return fmt.Errorf("missing invoice payload on notification event %s", e.ID)
		}

		return event.Payload.FromNotificationEventInvoiceCreatedPayload(api.NotificationEventInvoiceCreatedPayload{
			Id:        e.ID,
			Type:      api.NotificationEventInvoiceCreatedPayloadTypeV1InvoiceCreated,
			Timestamp: e.CreatedAt,
			Data:      toAPIInvoiceData(e.Payload.Invoice.Invoice),
		})

	case notification.EventTypeInvoiceUpdated:
		if e.Payload.Invoice == nil {
			return fmt.Errorf("missing invoice payload on notification event %s", e.ID)
		}

		return event.Payload.FromNotificationEventInvoiceUpdatedPayload(api.NotificationEventInvoiceUpdatedPayload{
			Id:        e.ID,
			Type:      api.NotificationEventInvoiceUpdatedPayloadTypeV1InvoiceUpdated,
			Timestamp: e.CreatedAt,
			Data:      toAPIInvoiceData(e.Payload.Invoice.Invoice),
		})

	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid notification event type: %s", e.Type))
	}
}

func toAPIEntitlementValue(v v1api.EntitlementValue) api.NotificationEventEntitlementValue {
	return api.NotificationEventEntitlementValue{
		HasAccess: v.HasAccess,
		Balance:   v.Balance,
		Usage:     v.Usage,
		Overage:   v.Overage,
	}
}

func toAPIInvoiceData(invoice v1api.Invoice) api.NotificationEventInvoiceData {
	return api.NotificationEventInvoiceData{
		Invoice: api.NotificationEventInvoiceReference{
			Id:     invoice.Id,
			Number: invoice.Number,
		},
		CustomerId: invoice.Customer.Id,
		Currency:   invoice.Currency,
		Status:     string(invoice.Status),
		Total:      invoice.Totals.Total,
	}
}

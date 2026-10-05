package events

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
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
func ToDomainDeliveryState(v api.BillingNotificationEventDeliveryState) (notification.EventDeliveryStatusState, error) {
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

func ToAPIDeliveryState(v notification.EventDeliveryStatusState) (api.BillingNotificationEventDeliveryState, error) {
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
func ToAPIBalanceThresholdType(v v1api.NotificationRuleBalanceThresholdValueType) (api.BillingNotificationBalanceThresholdType, error) {
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

func ToAPIEvent(e notification.Event) (api.BillingNotificationEvent, error) {
	deliveryStatus, err := ToAPIDeliveryStatuses(e.DeliveryStatus)
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
func ToAPIDeliveryStatuses(statuses []notification.EventDeliveryStatus) ([]api.BillingNotificationEventDeliveryStatus, error) {
	result := make([]api.BillingNotificationEventDeliveryStatus, 0, len(statuses))

	for _, status := range statuses {
		state, err := ToAPIDeliveryState(status.State)
		if err != nil {
			return nil, err
		}

		attempts, err := ToAPIDeliveryAttempts(status.Attempts)
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

func ToAPIDeliveryAttempts(attempts []notification.EventDeliveryAttempt) ([]api.BillingNotificationEventDeliveryAttempt, error) {
	notification.SortEventDeliveryAttemptsInDescOrder(attempts)

	result := make([]api.BillingNotificationEventDeliveryAttempt, 0, len(attempts))

	for _, attempt := range attempts {
		state, err := ToAPIDeliveryState(attempt.State)
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

// The stored payload holds v1 API models; only the identifiers and scalars the v3
// surface exposes are extracted, never the v1 shapes.
func setAPIEventPayload(event *api.BillingNotificationEvent, e notification.Event) error {
	switch e.Type {
	case notification.EventTypeBalanceThreshold:
		if e.Payload.BalanceThreshold == nil {
			return fmt.Errorf("missing balance threshold payload on notification event %s", e.ID)
		}

		thresholdType, err := ToAPIBalanceThresholdType(e.Payload.BalanceThreshold.Threshold.Type)
		if err != nil {
			return err
		}

		return event.Payload.FromBillingNotificationEventBalanceThresholdPayload(api.BillingNotificationEventBalanceThresholdPayload{
			Id:        e.ID,
			Type:      api.BillingNotificationEventBalanceThresholdPayloadTypeEntitlementsBalanceThreshold,
			Timestamp: e.CreatedAt,
			Data: api.BillingNotificationEventBalanceThresholdData{
				EntitlementId: e.Payload.BalanceThreshold.Entitlement.Id,
				Feature: api.BillingNotificationEventFeatureReference{
					Id:  e.Payload.BalanceThreshold.Feature.Id,
					Key: e.Payload.BalanceThreshold.Feature.Key,
				},
				SubjectKey: e.Payload.BalanceThreshold.Subject.Key,
				CustomerId: lo.EmptyableToPtr(e.Payload.BalanceThreshold.Customer.Id),
				Value:      toAPIEntitlementValue(e.Payload.BalanceThreshold.Feature.Key, e.Payload.BalanceThreshold.Value),
				Threshold: api.BillingNotificationBalanceThreshold{
					Type:  thresholdType,
					Value: e.Payload.BalanceThreshold.Threshold.Value,
				},
			},
		})

	case notification.EventTypeEntitlementReset:
		if e.Payload.EntitlementReset == nil {
			return fmt.Errorf("missing entitlement reset payload on notification event %s", e.ID)
		}

		return event.Payload.FromBillingNotificationEventResetPayload(api.BillingNotificationEventResetPayload{
			Id:        e.ID,
			Type:      api.BillingNotificationEventResetPayloadTypeEntitlementsReset,
			Timestamp: e.CreatedAt,
			Data: api.BillingNotificationEventEntitlementData{
				EntitlementId: e.Payload.EntitlementReset.Entitlement.Id,
				Feature: api.BillingNotificationEventFeatureReference{
					Id:  e.Payload.EntitlementReset.Feature.Id,
					Key: e.Payload.EntitlementReset.Feature.Key,
				},
				SubjectKey: e.Payload.EntitlementReset.Subject.Key,
				CustomerId: lo.EmptyableToPtr(e.Payload.EntitlementReset.Customer.Id),
				Value:      toAPIEntitlementValue(e.Payload.EntitlementReset.Feature.Key, e.Payload.EntitlementReset.Value),
			},
		})

	case notification.EventTypeInvoiceCreated:
		if e.Payload.Invoice == nil {
			return fmt.Errorf("missing invoice payload on notification event %s", e.ID)
		}

		return event.Payload.FromBillingNotificationEventInvoiceCreatedPayload(api.BillingNotificationEventInvoiceCreatedPayload{
			Id:        e.ID,
			Type:      api.BillingNotificationEventInvoiceCreatedPayloadTypeInvoiceCreated,
			Timestamp: e.CreatedAt,
			Data:      toAPIInvoiceData(e.Payload.Invoice.Invoice),
		})

	case notification.EventTypeInvoiceUpdated:
		if e.Payload.Invoice == nil {
			return fmt.Errorf("missing invoice payload on notification event %s", e.ID)
		}

		return event.Payload.FromBillingNotificationEventInvoiceUpdatedPayload(api.BillingNotificationEventInvoiceUpdatedPayload{
			Id:        e.ID,
			Type:      api.BillingNotificationEventInvoiceUpdatedPayloadTypeInvoiceUpdated,
			Timestamp: e.CreatedAt,
			Data:      toAPIInvoiceData(e.Payload.Invoice.Invoice),
		})

	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid notification event type: %s", e.Type))
	}
}

func toAPIEntitlementValue(featureKey string, v v1api.EntitlementValue) api.BillingEntitlementValueResult {
	return api.BillingEntitlementValueResult{
		Type:       api.BillingEntitlementTypeMetered,
		FeatureKey: featureKey,
		HasAccess:  v.HasAccess,
		Config:     v.Config,
		Value: &api.BillingEntitlementAccessValue{
			Balance:                   alpacadecimal.NewFromFloat(lo.FromPtr(v.Balance)).String(),
			Usage:                     alpacadecimal.NewFromFloat(lo.FromPtr(v.Usage)).String(),
			Overage:                   alpacadecimal.NewFromFloat(lo.FromPtr(v.Overage)).String(),
			TotalAvailableGrantAmount: alpacadecimal.NewFromFloat(lo.FromPtr(v.TotalAvailableGrantAmount)).String(),
			// keep it empty to satisfy the api contract
			GrantBalances: make(map[string]api.Numeric),
		},
	}
}

func toAPIInvoiceData(invoice v1api.Invoice) api.BillingNotificationEventInvoiceData {
	return api.BillingNotificationEventInvoiceData{
		Invoice: api.BillingNotificationEventInvoiceReference{
			Id:     invoice.Id,
			Number: invoice.Number,
		},
		CustomerId: invoice.Customer.Id,
		Currency:   invoice.Currency,
		Status:     string(invoice.Status),
		Total:      invoice.Totals.Total,
	}
}

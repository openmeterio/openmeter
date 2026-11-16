package events

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
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
func FromAPIBillingNotificationEventDeliveryState(v apiv3models.BillingNotificationEventDeliveryState) (notification.EventDeliveryStatusState, error) {
	switch v {
	case apiv3models.BillingNotificationEventDeliveryStateSuccess:
		return notification.EventDeliveryStatusStateSuccess, nil
	case apiv3models.BillingNotificationEventDeliveryStateFailed:
		return notification.EventDeliveryStatusStateFailed, nil
	case apiv3models.BillingNotificationEventDeliveryStateSending:
		return notification.EventDeliveryStatusStateSending, nil
	case apiv3models.BillingNotificationEventDeliveryStatePending:
		return notification.EventDeliveryStatusStatePending, nil
	case apiv3models.BillingNotificationEventDeliveryStateResending:
		return notification.EventDeliveryStatusStateResending, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid notification event delivery state: %s", v))
	}
}

func ToAPIBillingNotificationEventDeliveryState(v notification.EventDeliveryStatusState) (apiv3models.BillingNotificationEventDeliveryState, error) {
	switch v {
	case notification.EventDeliveryStatusStateSuccess:
		return apiv3models.BillingNotificationEventDeliveryStateSuccess, nil
	case notification.EventDeliveryStatusStateFailed:
		return apiv3models.BillingNotificationEventDeliveryStateFailed, nil
	case notification.EventDeliveryStatusStateSending:
		return apiv3models.BillingNotificationEventDeliveryStateSending, nil
	case notification.EventDeliveryStatusStatePending:
		return apiv3models.BillingNotificationEventDeliveryStatePending, nil
	case notification.EventDeliveryStatusStateResending:
		return apiv3models.BillingNotificationEventDeliveryStateResending, nil
	default:
		return "", fmt.Errorf("invalid notification event delivery state: %s", v)
	}
}

// The stored value is the v1 API model, where `NUMBER` and `PERCENT` are deprecated
// aliases of `usage_value` and `usage_percentage`; events written before the rename
// must not leak the legacy spelling.
func ToAPIBillingNotificationBalanceThresholdType(v apiv1models.NotificationRuleBalanceThresholdValueType) (apiv3models.BillingNotificationBalanceThresholdType, error) {
	switch v {
	case apiv1models.NotificationRuleBalanceThresholdValueTypeBalanceValue:
		return apiv3models.BillingNotificationBalanceThresholdTypeBalanceValue, nil
	case apiv1models.NotificationRuleBalanceThresholdValueTypeUsagePercentage,
		apiv1models.NotificationRuleBalanceThresholdValueTypePercent:
		return apiv3models.BillingNotificationBalanceThresholdTypeUsagePercentage, nil
	case apiv1models.NotificationRuleBalanceThresholdValueTypeUsageValue,
		apiv1models.NotificationRuleBalanceThresholdValueTypeNumber:
		return apiv3models.BillingNotificationBalanceThresholdTypeUsageValue, nil
	default:
		return "", fmt.Errorf("invalid notification balance threshold type: %s", v)
	}
}

func ToAPIBillingNotificationEvent(e notification.Event) (apiv3models.BillingNotificationEvent, error) {
	deliveryStatus, err := ToAPIBillingNotificationEventDeliveryStatuses(e.DeliveryStatus)
	if err != nil {
		return apiv3models.BillingNotificationEvent{}, err
	}

	event := apiv3models.BillingNotificationEvent{
		Id:        e.ID,
		Type:      apiv3models.BillingNotificationEventType(e.Type),
		CreatedAt: e.CreatedAt,
		Rule: apiv3models.BillingNotificationRuleReference{
			Id:   e.Rule.ID,
			Type: apiv3models.BillingNotificationEventType(e.Rule.Type),
			Name: e.Rule.Name,
		},
		DeliveryStatus: deliveryStatus,
	}

	if err := setAPIEventPayload(&event, e); err != nil {
		return apiv3models.BillingNotificationEvent{}, err
	}

	return event, nil
}

// Unlike v1, the channel is reported by id only, so a channel that has since been
// disabled or deleted is still reported.
func ToAPIBillingNotificationEventDeliveryStatuses(statuses []notification.EventDeliveryStatus) ([]apiv3models.BillingNotificationEventDeliveryStatus, error) {
	result := make([]apiv3models.BillingNotificationEventDeliveryStatus, 0, len(statuses))

	for _, status := range statuses {
		state, err := ToAPIBillingNotificationEventDeliveryState(status.State)
		if err != nil {
			return nil, err
		}

		attempts, err := ToAPIBillingNotificationEventDeliveryAttempts(status.Attempts)
		if err != nil {
			return nil, err
		}

		result = append(result, apiv3models.BillingNotificationEventDeliveryStatus{
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

func ToAPIBillingNotificationEventDeliveryAttempts(attempts []notification.EventDeliveryAttempt) ([]apiv3models.BillingNotificationEventDeliveryAttempt, error) {
	notification.SortEventDeliveryAttemptsInDescOrder(attempts)

	result := make([]apiv3models.BillingNotificationEventDeliveryAttempt, 0, len(attempts))

	for _, attempt := range attempts {
		state, err := ToAPIBillingNotificationEventDeliveryState(attempt.State)
		if err != nil {
			return nil, err
		}

		var statusCode *int32
		if attempt.Response.StatusCode != nil {
			statusCode = lo.ToPtr(int32(*attempt.Response.StatusCode))
		}

		result = append(result, apiv3models.BillingNotificationEventDeliveryAttempt{
			State:     state,
			Timestamp: attempt.Timestamp,
			Response: apiv3models.BillingNotificationEventDeliveryAttemptResponse{
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
func setAPIEventPayload(event *apiv3models.BillingNotificationEvent, e notification.Event) error {
	switch e.Type {
	case notification.EventTypeBalanceThreshold:
		if e.Payload.BalanceThreshold == nil {
			return fmt.Errorf("missing balance threshold payload on notification event %s", e.ID)
		}

		thresholdType, err := ToAPIBillingNotificationBalanceThresholdType(e.Payload.BalanceThreshold.Threshold.Type)
		if err != nil {
			return err
		}

		return event.Payload.FromBillingNotificationEventBalanceThresholdPayload(apiv3models.BillingNotificationEventBalanceThresholdPayload{
			Id:        e.ID,
			Type:      apiv3models.BillingNotificationEventBalanceThresholdPayloadTypeEntitlementsBalanceThreshold,
			Timestamp: e.CreatedAt,
			Data: apiv3models.BillingNotificationEventBalanceThresholdData{
				EntitlementId: e.Payload.BalanceThreshold.Entitlement.Id,
				Feature: apiv3models.BillingNotificationEventFeatureReference{
					Id:  e.Payload.BalanceThreshold.Feature.Id,
					Key: e.Payload.BalanceThreshold.Feature.Key,
				},
				SubjectKey: e.Payload.BalanceThreshold.Subject.Key,
				CustomerId: lo.EmptyableToPtr(e.Payload.BalanceThreshold.Customer.Id),
				Value:      toAPIEntitlementValue(e.Payload.BalanceThreshold.Feature.Key, e.Payload.BalanceThreshold.Value),
				Threshold: apiv3models.BillingNotificationBalanceThreshold{
					Type:  thresholdType,
					Value: e.Payload.BalanceThreshold.Threshold.Value,
				},
			},
		})

	case notification.EventTypeEntitlementReset:
		if e.Payload.EntitlementReset == nil {
			return fmt.Errorf("missing entitlement reset payload on notification event %s", e.ID)
		}

		return event.Payload.FromBillingNotificationEventResetPayload(apiv3models.BillingNotificationEventResetPayload{
			Id:        e.ID,
			Type:      apiv3models.BillingNotificationEventResetPayloadTypeEntitlementsReset,
			Timestamp: e.CreatedAt,
			Data: apiv3models.BillingNotificationEventEntitlementData{
				EntitlementId: e.Payload.EntitlementReset.Entitlement.Id,
				Feature: apiv3models.BillingNotificationEventFeatureReference{
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

		return event.Payload.FromBillingNotificationEventInvoiceCreatedPayload(apiv3models.BillingNotificationEventInvoiceCreatedPayload{
			Id:        e.ID,
			Type:      apiv3models.BillingNotificationEventInvoiceCreatedPayloadTypeInvoiceCreated,
			Timestamp: e.CreatedAt,
			Data:      toAPIInvoiceData(e.Payload.Invoice.Invoice),
		})

	case notification.EventTypeInvoiceUpdated:
		if e.Payload.Invoice == nil {
			return fmt.Errorf("missing invoice payload on notification event %s", e.ID)
		}

		return event.Payload.FromBillingNotificationEventInvoiceUpdatedPayload(apiv3models.BillingNotificationEventInvoiceUpdatedPayload{
			Id:        e.ID,
			Type:      apiv3models.BillingNotificationEventInvoiceUpdatedPayloadTypeInvoiceUpdated,
			Timestamp: e.CreatedAt,
			Data:      toAPIInvoiceData(e.Payload.Invoice.Invoice),
		})

	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid notification event type: %s", e.Type))
	}
}

func toAPIEntitlementValue(featureKey string, v apiv1models.EntitlementValue) apiv3models.BillingEntitlementValueResult {
	return apiv3models.BillingEntitlementValueResult{
		Type:       apiv3models.BillingEntitlementTypeMetered,
		FeatureKey: featureKey,
		HasAccess:  v.HasAccess,
		Config:     v.Config,
		Value: &apiv3models.BillingEntitlementAccessValue{
			Balance:                   alpacadecimal.NewFromFloat(lo.FromPtr(v.Balance)).String(),
			Usage:                     alpacadecimal.NewFromFloat(lo.FromPtr(v.Usage)).String(),
			Overage:                   alpacadecimal.NewFromFloat(lo.FromPtr(v.Overage)).String(),
			TotalAvailableGrantAmount: alpacadecimal.NewFromFloat(lo.FromPtr(v.TotalAvailableGrantAmount)).String(),
			// keep it empty to satisfy the api contract
			GrantBalances: make(map[string]apiv3models.Numeric),
		},
	}
}

func toAPIInvoiceData(invoice apiv1models.Invoice) apiv3models.BillingNotificationEventInvoiceData {
	return apiv3models.BillingNotificationEventInvoiceData{
		Invoice: apiv3models.BillingNotificationEventInvoiceReference{
			Id:     invoice.Id,
			Number: invoice.Number,
		},
		CustomerId: invoice.Customer.Id,
		Currency:   invoice.Currency,
		Status:     string(invoice.Status),
		Total:      invoice.Totals.Total,
	}
}

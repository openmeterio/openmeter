package events

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	v1api "github.com/openmeterio/openmeter/api"
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FromAPIEventSortField maps a v3 API sort field name to the domain OrderBy. The
// allow-list matches the orderings the adapter can actually apply
// (openmeter/notification/adapter/event.go); anything else would fall through to an
// order-by-id that the caller did not ask for.
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

// ToDomainEventType maps the v3 wire-format event type ("invoice_created") to the domain
// EventType ("invoice.created"). The wire value is snake_case to satisfy the v3 enum
// casing convention while the domain/DB value keeps the dotted form used by v1 and by
// the event bus. An unknown value returns a validation error rather than passing
// through, since an invalid EventType would silently match zero rows in filter[type].
func ToDomainEventType(v api.NotificationEventType) (notification.EventType, error) {
	switch v {
	case api.NotificationEventTypeEntitlementsBalanceThreshold:
		return notification.EventTypeBalanceThreshold, nil
	case api.NotificationEventTypeEntitlementsReset:
		return notification.EventTypeEntitlementReset, nil
	case api.NotificationEventTypeInvoiceCreated:
		return notification.EventTypeInvoiceCreated, nil
	case api.NotificationEventTypeInvoiceUpdated:
		return notification.EventTypeInvoiceUpdated, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid notification event type: %s", v))
	}
}

// ToAPIEventType maps the domain EventType to the v3 wire-format event type.
func ToAPIEventType(v notification.EventType) (api.NotificationEventType, error) {
	switch v {
	case notification.EventTypeBalanceThreshold:
		return api.NotificationEventTypeEntitlementsBalanceThreshold, nil
	case notification.EventTypeEntitlementReset:
		return api.NotificationEventTypeEntitlementsReset, nil
	case notification.EventTypeInvoiceCreated:
		return api.NotificationEventTypeInvoiceCreated, nil
	case notification.EventTypeInvoiceUpdated:
		return api.NotificationEventTypeInvoiceUpdated, nil
	default:
		return "", fmt.Errorf("invalid notification event type: %s", v)
	}
}

// ToDomainDeliveryState maps the v3 wire-format delivery state ("failed") to the domain
// state ("FAILED"). As with the event type, the wire value is lowercased for the v3 enum
// casing convention while the column keeps the uppercase value written by v1.
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

// ToAPIDeliveryState maps the domain delivery state to its v3 wire-format value.
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

// ToAPIBalanceThresholdType maps a stored threshold type to its v3 value. The stored
// value comes from the v1 API model, where `NUMBER` and `PERCENT` are deprecated aliases
// of `usage_value` and `usage_percentage`; normalizing them here means events written
// before the rename report the current value instead of leaking the legacy spelling.
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

// mapAPIEnumFilter rewrites the wire values of an exact-match filter into their domain
// equivalents so the predicate matches what is actually stored in the column.
// filters.FromAPIFilterStringExact and filters.FromAPIFilterULID always produce a single
// flat *filter.FilterString for the operators this endpoint accepts (never And-wrapped),
// so translating Eq and In covers every value that can reach the adapter.
func mapAPIEnumFilter(f *filter.FilterString, mapValue func(string) (string, error)) (*filter.FilterString, error) {
	if f == nil {
		return nil, nil
	}

	mapped := *f

	if f.Eq != nil {
		v, err := mapValue(*f.Eq)
		if err != nil {
			return nil, err
		}
		mapped.Eq = lo.ToPtr(v)
	}

	if f.In != nil {
		values := make([]string, 0, len(*f.In))
		for _, raw := range *f.In {
			v, err := mapValue(raw)
			if err != nil {
				return nil, err
			}
			values = append(values, v)
		}
		mapped.In = &values
	}

	return &mapped, nil
}

// requireExactFilter rejects operators the backing predicate cannot express faithfully.
// The channel and delivery-state filters are existential joins, so a negated predicate
// means "has some other channel/state as well" rather than "does not have this one";
// subject and feature live in a JSONB annotation document reachable only through
// entutils.JSONBIn. In both cases a non-exact operator would return rows that do not
// answer the question that was asked, so it fails loudly instead.
func requireExactFilter(field string, f *filter.FilterString) error {
	if f == nil || f.IsEmpty() {
		return nil
	}

	if f.Eq != nil || f.In != nil {
		return nil
	}

	return fmt.Errorf("filter[%s] only supports the eq and oeq operators", field)
}

// ToAPIEvent maps a domain Event to its v3 API representation.
func ToAPIEvent(e notification.Event) (api.NotificationEvent, error) {
	eventType, err := ToAPIEventType(e.Type)
	if err != nil {
		return api.NotificationEvent{}, err
	}

	ruleType, err := ToAPIEventType(e.Rule.Type)
	if err != nil {
		return api.NotificationEvent{}, fmt.Errorf("failed to map notification rule type: %w", err)
	}

	deliveryStatus, err := ToAPIDeliveryStatuses(e.DeliveryStatus)
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

// ToAPIDeliveryStatuses maps the per-channel delivery statuses of an event. Unlike v1,
// the channel is reported by id only, so no lookup into the rule's channel list is
// needed and a channel that has since been disabled or deleted is still reported
// correctly.
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

// ToAPIDeliveryAttempts maps the delivery attempts of a single channel, most recent
// first.
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

// setAPIEventPayload fills the payload union of an API event from the stored domain
// payload. The stored payload holds v1 API models, so this extracts the identifiers and
// scalars the v3 surface exposes instead of re-emitting the v1 shapes.
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
			Type:      api.NotificationEventBalanceThresholdPayloadTypeEntitlementsBalanceThreshold,
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
			Type:      api.NotificationEventResetPayloadTypeEntitlementsReset,
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
			Type:      api.NotificationEventInvoiceCreatedPayloadTypeInvoiceCreated,
			Timestamp: e.CreatedAt,
			Data:      toAPIInvoiceData(e.Payload.Invoice.Invoice),
		})

	case notification.EventTypeInvoiceUpdated:
		if e.Payload.Invoice == nil {
			return fmt.Errorf("missing invoice payload on notification event %s", e.ID)
		}

		return event.Payload.FromNotificationEventInvoiceUpdatedPayload(api.NotificationEventInvoiceUpdatedPayload{
			Id:        e.ID,
			Type:      api.NotificationEventInvoiceUpdatedPayloadTypeInvoiceUpdated,
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

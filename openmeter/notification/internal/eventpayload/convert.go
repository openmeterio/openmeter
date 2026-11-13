package eventpayload

import (
	"fmt"

	"github.com/samber/lo"

	apimodels "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

func ToAPINotificationEventBalanceThresholdPayload(e notification.Event) (apimodels.NotificationEventBalanceThresholdPayload, error) {
	if e.Payload.BalanceThreshold == nil {
		return apimodels.NotificationEventBalanceThresholdPayload{}, fmt.Errorf("balance threshold is nil")
	}

	return apimodels.NotificationEventBalanceThresholdPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Type:      apimodels.NotificationEventBalanceThresholdPayloadTypeEntitlementsBalanceThreshold,
		Data: apimodels.NotificationEventBalanceThresholdPayloadData{
			Value:       e.Payload.BalanceThreshold.Value,
			Entitlement: e.Payload.BalanceThreshold.Entitlement,
			Feature:     e.Payload.BalanceThreshold.Feature,
			Subject:     e.Payload.BalanceThreshold.Subject,
			Customer:    lo.ToPtr(e.Payload.BalanceThreshold.Customer),
			Threshold:   e.Payload.BalanceThreshold.Threshold,
		},
	}, nil
}

func ToAPINotificationEventResetPayload(e notification.Event) (apimodels.NotificationEventResetPayload, error) {
	if e.Payload.EntitlementReset == nil {
		return apimodels.NotificationEventResetPayload{}, fmt.Errorf("entitlement reset is nil")
	}

	return apimodels.NotificationEventResetPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Type:      apimodels.NotificationEventResetPayloadTypeEntitlementsReset,
		Data: apimodels.NotificationEventEntitlementValuePayloadBase{
			Value:       e.Payload.EntitlementReset.Value,
			Entitlement: e.Payload.EntitlementReset.Entitlement,
			Feature:     e.Payload.EntitlementReset.Feature,
			Subject:     e.Payload.EntitlementReset.Subject,
			Customer:    lo.ToPtr(e.Payload.EntitlementReset.Customer),
		},
	}, nil
}

func ToAPINotificationEventInvoiceCreatedPayload(e notification.Event) (apimodels.NotificationEventInvoiceCreatedPayload, error) {
	if e.Payload.Invoice == nil {
		return apimodels.NotificationEventInvoiceCreatedPayload{}, fmt.Errorf("invoice is nil")
	}

	return apimodels.NotificationEventInvoiceCreatedPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Data:      e.Payload.Invoice.Invoice,
		Type:      apimodels.NotificationEventInvoiceCreatedPayloadTypeInvoiceCreated,
	}, nil
}

func ToAPINotificationEventInvoiceUpdatedPayload(e notification.Event) (apimodels.NotificationEventInvoiceUpdatedPayload, error) {
	if e.Payload.Invoice == nil {
		return apimodels.NotificationEventInvoiceUpdatedPayload{}, fmt.Errorf("invoice is nil")
	}

	return apimodels.NotificationEventInvoiceUpdatedPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Data:      e.Payload.Invoice.Invoice,
		Type:      apimodels.NotificationEventInvoiceUpdatedPayloadTypeInvoiceUpdated,
	}, nil
}

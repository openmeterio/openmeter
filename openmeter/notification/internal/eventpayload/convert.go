package eventpayload

import (
	"fmt"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

func ToAPINotificationEventBalanceThresholdPayload(e notification.Event) (apiv1models.NotificationEventBalanceThresholdPayload, error) {
	if e.Payload.BalanceThreshold == nil {
		return apiv1models.NotificationEventBalanceThresholdPayload{}, fmt.Errorf("balance threshold is nil")
	}

	return apiv1models.NotificationEventBalanceThresholdPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Type:      apiv1models.NotificationEventBalanceThresholdPayloadTypeEntitlementsBalanceThreshold,
		Data: apiv1models.NotificationEventBalanceThresholdPayloadData{
			Value:       e.Payload.BalanceThreshold.Value,
			Entitlement: e.Payload.BalanceThreshold.Entitlement,
			Feature:     e.Payload.BalanceThreshold.Feature,
			Subject:     e.Payload.BalanceThreshold.Subject,
			Customer:    lo.ToPtr(e.Payload.BalanceThreshold.Customer),
			Threshold:   e.Payload.BalanceThreshold.Threshold,
		},
	}, nil
}

func ToAPINotificationEventResetPayload(e notification.Event) (apiv1models.NotificationEventResetPayload, error) {
	if e.Payload.EntitlementReset == nil {
		return apiv1models.NotificationEventResetPayload{}, fmt.Errorf("entitlement reset is nil")
	}

	return apiv1models.NotificationEventResetPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Type:      apiv1models.NotificationEventResetPayloadTypeEntitlementsReset,
		Data: apiv1models.NotificationEventEntitlementValuePayloadBase{
			Value:       e.Payload.EntitlementReset.Value,
			Entitlement: e.Payload.EntitlementReset.Entitlement,
			Feature:     e.Payload.EntitlementReset.Feature,
			Subject:     e.Payload.EntitlementReset.Subject,
			Customer:    lo.ToPtr(e.Payload.EntitlementReset.Customer),
		},
	}, nil
}

func ToAPINotificationEventInvoiceCreatedPayload(e notification.Event) (apiv1models.NotificationEventInvoiceCreatedPayload, error) {
	if e.Payload.Invoice == nil {
		return apiv1models.NotificationEventInvoiceCreatedPayload{}, fmt.Errorf("invoice is nil")
	}

	return apiv1models.NotificationEventInvoiceCreatedPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Data:      e.Payload.Invoice.Invoice,
		Type:      apiv1models.NotificationEventInvoiceCreatedPayloadTypeInvoiceCreated,
	}, nil
}

func ToAPINotificationEventInvoiceUpdatedPayload(e notification.Event) (apiv1models.NotificationEventInvoiceUpdatedPayload, error) {
	if e.Payload.Invoice == nil {
		return apiv1models.NotificationEventInvoiceUpdatedPayload{}, fmt.Errorf("invoice is nil")
	}

	return apiv1models.NotificationEventInvoiceUpdatedPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Data:      e.Payload.Invoice.Invoice,
		Type:      apiv1models.NotificationEventInvoiceUpdatedPayloadTypeInvoiceUpdated,
	}, nil
}

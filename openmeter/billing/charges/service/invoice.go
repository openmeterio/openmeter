package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) handleStandardInvoiceUpdate(ctx context.Context, invoice billing.StandardInvoice) error {
	switch invoice.Status {
	case billing.StandardInvoiceStatusDraftCreated,
		billing.StandardInvoiceStatusIssued,
		billing.StandardInvoiceStatusPaymentProcessingAuthorized,
		billing.StandardInvoiceStatusPaymentProcessingBookingAuthorizedAndSettled,
		billing.StandardInvoiceStatusPaid:
	default:
		return nil
	}

	fiatCurrency, err := invoice.Currency.AsFiatCurrency()
	if err != nil {
		return fmt.Errorf("resolving fiat invoice currency %q: %w", invoice.Currency, err)
	}
	currency := currencies.Currency{Currency: fiatCurrency}

	return s.recognizeCustomerEarnings(ctx, invoice.CustomerID(), currency)
}

var _ billing.StandardInvoiceHook = (*standardInvoiceEventHandler)(nil)

// standardInvoiceEventHandler feeds invoice lifecycle changes into earnings recognition.
// Charge lifecycle transitions are owned by their line engines.
type standardInvoiceEventHandler struct {
	models.NoopServiceHook[billing.StandardInvoice]
	chargesService *service
}

func (h *standardInvoiceEventHandler) PostUpdate(ctx context.Context, invoice *billing.StandardInvoice) error {
	return h.chargesService.handleStandardInvoiceUpdate(ctx, *invoice)
}

func (h *standardInvoiceEventHandler) PostCreate(ctx context.Context, invoice *billing.StandardInvoice) error {
	// The creation can be treated as an update from out perspective for the draft.created state.
	return h.chargesService.handleStandardInvoiceUpdate(ctx, *invoice)
}

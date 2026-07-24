package service

import (
	"context"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPostInvoicePaymentAuthorizedPersistsInvoiceFiatAmount(t *testing.T) {
	// given:
	// - a credit purchase whose credit amount differs from its final invoice total
	// when:
	// - the invoice payment is authorized
	// then:
	// - the payment realization stores the invoice total in fiat currency
	charge := newExternalStateMachineTestCharge(t, creditpurchase.StatusActive, alpacadecimal.NewFromFloat(0.5))
	adapter := &invoicePaymentAdapter{}
	handler := &invoicePaymentHandler{}
	handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
		Once()

	service := &service{
		adapter: adapter,
		handler: handler,
	}

	lineWithHeader := billing.StandardLineWithInvoiceHeader{
		Line: &billing.StandardLine{
			StandardLineBase: billing.StandardLineBase{
				ManagedResource: models.ManagedResource{
					ID: "line-1",
				},
				Totals: totals.Totals{
					Total: alpacadecimal.NewFromFloat(50.06),
				},
			},
		},
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				ID: "invoice-1",
			},
		},
	}

	err := service.PostInvoicePaymentAuthorized(t.Context(), charge, lineWithHeader)

	require.NoError(t, err)
	require.Equal(t, float64(50.06), adapter.createdPayment.FiatAmount.InexactFloat64())
	require.Equal(t, "invoice-1", adapter.createdPayment.InvoiceID)
	require.Equal(t, "line-1", adapter.createdPayment.LineID)
	handler.AssertExpectations(t)
}

type invoicePaymentAdapter struct {
	creditpurchase.Adapter

	createdPayment payment.InvoicedCreate
}

func (a *invoicePaymentAdapter) CreateInvoicedPayment(_ context.Context, _ meta.ChargeID, input payment.InvoicedCreate) (payment.Invoiced, error) {
	a.createdPayment = input
	return payment.Invoiced{}, nil
}

type invoicePaymentHandler struct {
	creditpurchase.Handler
	mock.Mock
}

func (h *invoicePaymentHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, input creditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	args := h.Called(ctx, input)
	return args.Get(0).(ledgertransaction.GroupReference), args.Error(1)
}

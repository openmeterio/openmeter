package service

import (
	"context"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestPostInvoicePaymentAuthorizedUsesCreditAmount(t *testing.T) {
	charge := newExternalStateMachineTestChargeWithInput(externalStateMachineTestChargeInput{
		status:        creditpurchase.StatusActive,
		currency:      currencyx.Code("ACME"),
		costBasis:     alpacadecimal.NewFromFloat(0.5),
		creditAmount:  alpacadecimal.NewFromInt(10),
		initialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
	})
	charge.Intent.Settlement = creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{
		GenericSettlement: creditpurchase.GenericSettlement{
			Currency:  currencyx.Code("USD"),
			CostBasis: alpacadecimal.NewFromFloat(0.5),
		},
	})

	adapter := &invoicePaymentAdapter{}
	handler := &invoicePaymentHandler{}
	service := &service{
		adapter: adapter,
		handler: handler,
	}
	line := &billing.StandardLine{}
	line.ID = "line-1"
	invoice := billing.StandardInvoice{}
	invoice.ID = "invoice-1"

	err := service.PostInvoicePaymentAuthorized(t.Context(), charge, billing.StandardLineWithInvoiceHeader{
		Line:    line,
		Invoice: invoice,
	})
	require.NoError(t, err)
	require.Equal(t, float64(10), adapter.created.Amount.InexactFloat64())
	require.Equal(t, "authorized-ledger-tx", adapter.created.Authorized.TransactionGroupID)
}

type invoicePaymentAdapter struct {
	creditpurchase.Adapter

	created payment.InvoicedCreate
}

func (a *invoicePaymentAdapter) CreateInvoicedPayment(_ context.Context, _ meta.ChargeID, input payment.InvoicedCreate) (payment.Invoiced, error) {
	a.created = input

	return payment.Invoiced{}, nil
}

type invoicePaymentHandler struct {
	creditpurchase.Handler
}

func (h *invoicePaymentHandler) OnCreditPurchasePaymentAuthorized(context.Context, creditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil
}

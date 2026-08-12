package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestBuildInvoiceCreditPurchaseGatheringLineUsesZeroForUnresolvedCostBasis(t *testing.T) {
	// given:
	// - an invoice credit purchase whose dynamic cost basis is not yet resolved
	// when:
	// - its gathering line is built
	// then:
	// - the line keeps the settlement currency and uses a temporary zero amount
	charge := newInvoiceStateMachineTestCharge(t, creditpurchase.StatusCreated)
	charge.Intent.Currency = currenciestestutils.NewCustomCurrency(t, "TOKENS", 3)
	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	charge.Intent.CostBasis = creditpurchase.NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.DynamicIntent{
		FiatCurrency: fiatCurrency,
	}))
	charge.State.ChargeCostBasisID = lo.ToPtr("charge-cost-basis-1")
	charge.State.ResolvedCostBasis = nil

	line, err := buildInvoiceCreditPurchaseGatheringLine(charge)
	require.NoError(t, err)
	require.Equal(t, currencyx.FiatCode("USD"), line.Currency)
	price, err := line.Price.AsFlat()
	require.NoError(t, err)
	require.Equal(t, float64(0), price.Amount.InexactFloat64())
}

func TestInvoiceCreditPurchaseStateMachineRejectsMismatchedPaymentLine(t *testing.T) {
	charge := newGrantedInvoiceStateMachineTestCharge(t, creditpurchase.StatusActivePaymentAuthorized)
	lineWithHeader := newInvoiceStateMachineTestLine(t, charge, alpacadecimal.NewFromInt(50))
	charge.Realizations.InvoiceSettlement = newAuthorizedInvoiceStateMachinePayment(charge, lineWithHeader)
	lineWithHeader.Line.ID = "different-line"
	adapter := &externalStateMachineAdapter{}
	handler := &externalStateMachineHandler{}
	service := &service{
		adapter:           adapter,
		realizations:      newExternalStateMachineRealizations(t, adapter, handler, &externalStateMachineLineage{}),
		costbasisResolver: externalStateMachineCostBasisResolver{},
	}

	_, err := service.handleInvoiceLifecycleTrigger(t.Context(), HandleInvoiceLifecycleTriggerInput{
		Charge:         charge,
		Trigger:        billing.TriggerPaid,
		LineWithHeader: lineWithHeader,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "payment line ID must match line")
	require.Zero(t, adapter.updateInvoicedPaymentCalls)
	require.Zero(t, adapter.updateChargeCalls)
}

func newInvoiceStateMachineTestCharge(t *testing.T, status creditpurchase.Status) creditpurchase.Charge {
	t.Helper()

	charge := newExternalStateMachineTestCharge(t, status, alpacadecimal.NewFromFloat(0.5))
	charge.Intent.Name = "test invoice credits"
	charge.Intent.Settlement = creditpurchase.NewInvoiceSettlement()

	return charge
}

func newGrantedInvoiceStateMachineTestCharge(t *testing.T, status creditpurchase.Status) creditpurchase.Charge {
	t.Helper()

	charge := newInvoiceStateMachineTestCharge(t, status)
	charge.Realizations.CreditGrantRealization = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "initiated-ledger-tx"},
		Time:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	return charge
}

func newInvoiceStateMachineTestLine(t *testing.T, charge creditpurchase.Charge, fiatAmount alpacadecimal.Decimal) billing.StandardLineWithInvoiceHeader {
	t.Helper()

	invoiceID := "invoice-1"
	chargeID := charge.ID
	createdAt := charge.Intent.ServicePeriod.From
	line := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: charge.Namespace},
				ManagedModel: models.ManagedModel{
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
				},
				ID:   "line-1",
				Name: "invoice credit purchase",
			},
			ManagedBy: billing.SystemManagedLine,
			Engine:    billing.LineEngineTypeChargeCreditPurchase,
			InvoiceID: invoiceID,
			Currency:  currencyx.FiatCode("USD"),
			Period:    charge.Intent.ServicePeriod,
			InvoiceAt: createdAt,
			ChargeID:  &chargeID,
			Totals: totals.Totals{
				Amount: fiatAmount,
				Total:  fiatAmount,
			},
		},
		UsageBased: &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      fiatAmount,
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Quantity: lo.ToPtr(alpacadecimal.NewFromInt(1)),
		},
	}

	lineWithHeader := billing.StandardLineWithInvoiceHeader{
		Line: line,
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				Namespace: charge.Namespace,
				ID:        invoiceID,
			},
		},
	}
	require.NoError(t, lineWithHeader.Validate())

	return lineWithHeader
}

func newAuthorizedInvoiceStateMachinePayment(charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) *payment.Invoiced {
	return &payment.Invoiced{
		Payment: payment.Payment{
			NamespacedID: models.NamespacedID{Namespace: charge.Namespace, ID: "invoice-payment-1"},
			ManagedModel: models.ManagedModel{
				CreatedAt: charge.Intent.ServicePeriod.From,
				UpdatedAt: charge.Intent.ServicePeriod.From,
			},
			Base: payment.Base{
				ServicePeriod: charge.Intent.ServicePeriod,
				Status:        payment.StatusAuthorized,
				FiatAmount:    lineWithHeader.Line.Totals.Total,
				Authorized: &ledgertransaction.TimedGroupReference{
					GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"},
					Time:           charge.Intent.ServicePeriod.From,
				},
			},
		},
		LineID:    lineWithHeader.Line.ID,
		InvoiceID: lineWithHeader.Invoice.ID,
	}
}

package run

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestBookInvoicedPaymentAuthorizedInputValidate(t *testing.T) {
	valid := newBookPaymentAuthorizedInput()
	require.NoError(t, valid.Validate())

	t.Run("rejects existing payment", func(t *testing.T) {
		in := newBookPaymentAuthorizedInput()
		in.Run.Payment = &payment.Invoiced{
			Payment: payment.Payment{
				NamespacedID: models.NamespacedID{Namespace: in.Charge.Namespace, ID: "payment-1"},
				ManagedModel: models.ManagedModel{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
				Base: payment.Base{
					ServicePeriod: in.Line.Period,
					Status:        payment.StatusAuthorized,
					Amount:        in.Line.Totals.Total,
					Authorized: &ledgertransaction.TimedGroupReference{
						GroupReference: ledgertransaction.GroupReference{
							TransactionGroupID: "authorized-group",
						},
						Time: time.Now().UTC(),
					},
				},
			},
			LineID:    in.Line.ID,
			InvoiceID: in.Invoice.ID,
		}
		require.ErrorContains(t, in.Validate(), "payment already authorized")
	})

	t.Run("rejects mismatched line id", func(t *testing.T) {
		in := newBookPaymentAuthorizedInput()
		other := "other-line"
		in.Run.LineID = &other
		require.ErrorContains(t, in.Validate(), "already linked to a different line")
	})
}

func TestSettleInvoicedPaymentInputValidate(t *testing.T) {
	valid := newSettlePaymentInput()
	require.NoError(t, valid.Validate())

	t.Run("rejects missing payment", func(t *testing.T) {
		in := newSettlePaymentInput()
		in.Run.Payment = nil
		require.ErrorContains(t, in.Validate(), "cannot settle an unauthorized payment")
	})

	t.Run("allows missing payment when no fiat transaction is required", func(t *testing.T) {
		in := newSettlePaymentInput()
		in.Run.Payment = nil
		in.Run.NoFiatTransactionRequired = true
		require.NoError(t, in.Validate())
	})

	t.Run("rejects mismatched payment line id", func(t *testing.T) {
		in := newSettlePaymentInput()
		in.Run.Payment.LineID = "other-line"
		require.ErrorContains(t, in.Validate(), "payment line ID does not match")
	})

	t.Run("rejects non authorized payment status", func(t *testing.T) {
		in := newSettlePaymentInput()
		in.Run.Payment.Status = payment.StatusSettled
		in.Run.Payment.Settled = &ledgertransaction.TimedGroupReference{
			GroupReference: ledgertransaction.GroupReference{
				TransactionGroupID: "settled-group",
			},
			Time: time.Now().UTC(),
		}
		require.ErrorContains(t, in.Validate(), "payment already settled")
	})
}

func newBookPaymentAuthorizedInput() BookInvoicedPaymentAuthorizedInput {
	lineID := "line-1"
	now := time.Now().UTC()
	return BookInvoicedPaymentAuthorizedInput{
		Charge: newUsageBasedCharge(),
		Run:    newUsageBasedRun(lineID),
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				Namespace: "ns",
				ID:        "invoice-1",
			},
		},
		Line: billing.StandardLine{
			StandardLineBase: billing.StandardLineBase{
				ManagedResource: models.ManagedResource{
					NamespacedModel: models.NamespacedModel{Namespace: "ns"},
					ManagedModel:    models.ManagedModel{CreatedAt: now, UpdatedAt: now},
					ID:              lineID,
					Name:            "line-1",
				},
				ManagedBy: billing.SystemManagedLine,
				Currency:  currencyx.Code("USD"),
				InvoiceID: "invoice-1",
				InvoiceAt: now,
				Period: timeutil.ClosedPeriod{
					From: now.Add(-time.Hour),
					To:   now,
				},
				Totals: totals.Totals{
					Amount: alpacadecimal.NewFromInt(10),
					Total:  alpacadecimal.NewFromInt(10),
				},
			},
			UsageBased: &billing.UsageBasedLine{
				Price:           productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				FeatureKey:      "api_requests",
				Quantity:        lo.ToPtr(alpacadecimal.NewFromInt(10)),
				MeteredQuantity: lo.ToPtr(alpacadecimal.NewFromInt(10)),
			},
		},
	}
}

func newSettlePaymentInput() SettleInvoicedPaymentInput {
	authInput := newBookPaymentAuthorizedInput()
	authInput.Run.InvoiceUsage = &invoicedusage.AccruedUsage{
		LineID:        &authInput.Line.ID,
		ServicePeriod: authInput.Line.Period,
		Mutable:       false,
		Totals:        authInput.Line.Totals,
	}
	authInput.Run.Payment = &payment.Invoiced{
		Payment: payment.Payment{
			NamespacedID: models.NamespacedID{Namespace: authInput.Charge.Namespace, ID: "payment-1"},
			ManagedModel: models.ManagedModel{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
			Base: payment.Base{
				ServicePeriod: authInput.Line.Period,
				Status:        payment.StatusAuthorized,
				Amount:        authInput.Line.Totals.Total,
				Authorized: &ledgertransaction.TimedGroupReference{
					GroupReference: ledgertransaction.GroupReference{
						TransactionGroupID: "authorized-group",
					},
					Time: time.Now().UTC(),
				},
			},
		},
		LineID:    authInput.Line.ID,
		InvoiceID: authInput.Invoice.ID,
	}

	return SettleInvoicedPaymentInput(authInput)
}

func newUsageBasedCharge() usagebased.Charge {
	now := time.Now().UTC()
	period := timeutil.ClosedPeriod{From: now.Add(-2 * time.Hour), To: now.Add(-time.Hour)}

	return usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ManagedModel:    models.ManagedModel{CreatedAt: now, UpdatedAt: now},
				ID:              "charge-1",
			},
			Intent: usagebased.Intent{
				Intent: meta.Intent{
					Name:          "usage based",
					ManagedBy:     billing.SystemManagedLine,
					CustomerID:    "cust-1",
					Currency:      currencyx.Code("USD"),
					ServicePeriod: period,
					BillingPeriod: period,
				},
				InvoiceAt:      now,
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				FeatureKey:     "api_requests",
				Price:          *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			},
			Status: usagebased.StatusActiveAwaitingPaymentSettlement,
			State: usagebased.State{
				FeatureID:    "feature-1",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
	}
}

func newUsageBasedRun(lineID string) usagebased.RealizationRun {
	now := time.Now().UTC()
	return usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID:              usagebased.RealizationRunID(models.NamespacedID{Namespace: "ns", ID: "run-1"}),
			ManagedModel:    models.ManagedModel{CreatedAt: now, UpdatedAt: now},
			FeatureID:       "feature-1",
			LineID:          &lineID,
			Type:            usagebased.RealizationRunTypeFinalRealization,
			InitialType:     usagebased.RealizationRunTypeFinalRealization,
			StoredAtLT:      now,
			ServicePeriodTo: now,
			MeteredQuantity: alpacadecimal.NewFromInt(10),
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(10),
				Total:  alpacadecimal.NewFromInt(10),
			},
		},
	}
}

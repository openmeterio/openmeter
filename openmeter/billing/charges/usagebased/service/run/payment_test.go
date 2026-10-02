package run

import (
	"context"
	"errors"
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
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestBookInvoicedPaymentAuthorizedInputValidate(t *testing.T) {
	valid := newBookPaymentAuthorizedInput(t)
	require.NoError(t, valid.Validate())

	t.Run("rejects mismatched line id", func(t *testing.T) {
		in := newBookPaymentAuthorizedInput(t)
		other := "other-line"
		in.Run.LineID = &other
		require.ErrorContains(t, in.Validate(), "already linked to a different line")
	})
}

func TestBookInvoicedPaymentAuthorizedSkipsZeroFiatAmount(t *testing.T) {
	in := newBookPaymentAuthorizedInput(t)
	in.Line.Totals = totals.Totals{}
	service := Service{handler: &usagebased.UnimplementedHandler{}}

	result, err := service.BookInvoicedPaymentAuthorized(t.Context(), in)
	require.NoError(t, err)
	require.Nil(t, result.Payment)
}

func TestSettleInvoicedPaymentInputValidate(t *testing.T) {
	valid := newSettlePaymentInput(t)
	require.NoError(t, valid.Validate())

	t.Run("rejects missing payment", func(t *testing.T) {
		in := newSettlePaymentInput(t)
		in.Run.Payment = nil
		require.ErrorContains(t, in.Validate(), "cannot settle an unauthorized payment")
	})

	t.Run("allows missing payment when no fiat transaction is required", func(t *testing.T) {
		in := newSettlePaymentInput(t)
		in.Run.Payment = nil
		in.Run.NoFiatTransactionRequired = true
		require.NoError(t, in.Validate())
	})

	t.Run("rejects mismatched payment line id", func(t *testing.T) {
		in := newSettlePaymentInput(t)
		in.Run.Payment.LineID = "other-line"
		require.ErrorContains(t, in.Validate(), "payment line ID does not match")
	})

	t.Run("rejects non authorized payment status", func(t *testing.T) {
		in := newSettlePaymentInput(t)
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

func TestSettleInvoicedPaymentSkipsZeroFiatAmount(t *testing.T) {
	in := newSettlePaymentInput(t)
	in.Line.Totals = totals.Totals{}
	service := Service{handler: &usagebased.UnimplementedHandler{}}

	result, err := service.SettleInvoicedPayment(t.Context(), in)
	require.NoError(t, err)
	require.Nil(t, result.Payment)
}

func newBookPaymentAuthorizedInput(t testing.TB) BookInvoicedPaymentAuthorizedInput {
	t.Helper()

	lineID := "line-1"
	now := time.Now().UTC()
	return BookInvoicedPaymentAuthorizedInput{
		Charge: newUsageBasedCharge(t),
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
				Currency:  currencyx.FiatCode("USD"),
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

func newSettlePaymentInput(t testing.TB) SettleInvoicedPaymentInput {
	t.Helper()

	authInput := newBookPaymentAuthorizedInput(t)
	authInput.Run.InvoiceUsage = &invoicedusage.AccruedUsage{
		ServicePeriod: authInput.Line.Period,
		Totals:        authInput.Line.Totals,
	}
	authInput.Run.Payment = &payment.Invoiced{
		Payment: payment.Payment{
			NamespacedID: models.NamespacedID{Namespace: authInput.Charge.Namespace, ID: "payment-1"},
			ManagedModel: models.ManagedModel{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
			Base: payment.Base{
				ServicePeriod: authInput.Line.Period,
				Status:        payment.StatusAuthorized,
				FiatAmount:    authInput.Line.Totals.Total,
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

func newUsageBasedCharge(t testing.TB) usagebased.Charge {
	t.Helper()

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
					ManagedBy:  billing.SystemManagedLine,
					CustomerID: "cust-1",
					Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: "tax-code-id",
					},
				},
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:          "usage based",
						ServicePeriod: period,
						BillingPeriod: period,
					},
					InvoiceAt: now,
					Price:     *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				FeatureKey:     "api_requests",
			}.AsOverridableIntent(),
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

func TestBookInvoicedPaymentAuthorizedRecognizesMatchingBooking(t *testing.T) {
	for _, tc := range []struct {
		name          string
		mutate        func(*payment.Invoiced)
		invoiceAmount *alpacadecimal.Decimal
		expectedError error
		invalidField  string
	}{
		{name: "matching authorization"},
		{
			name:          "zero invoice amount conflicts with existing authorization",
			invoiceAmount: lo.ToPtr(alpacadecimal.NewFromInt(0)),
			expectedError: payment.ErrPaymentAlreadyAuthorized,
		},
		{
			name: "matching settled payment",
			mutate: func(booked *payment.Invoiced) {
				booked.Status = payment.StatusSettled
				booked.Settled = &ledgertransaction.TimedGroupReference{
					GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "settlement-group"},
					Time:           booked.Authorized.Time,
				}
			},
		},
		{
			name: "different namespace", mutate: func(booked *payment.Invoiced) { booked.Namespace = "other-namespace" },
			expectedError: payment.ErrPaymentAlreadyAuthorized,
		},
		{
			name: "different invoice", mutate: func(booked *payment.Invoiced) { booked.InvoiceID = "other-invoice" },
			expectedError: payment.ErrPaymentAlreadyAuthorized,
		},
		{
			name: "different line", mutate: func(booked *payment.Invoiced) { booked.LineID = "other-line" },
			expectedError: payment.ErrPaymentAlreadyAuthorized,
		},
		{
			name: "different service period retains the same authorization", mutate: func(booked *payment.Invoiced) { booked.ServicePeriod.To = booked.ServicePeriod.To.Add(time.Hour) },
		},
		{
			name: "different amount", mutate: func(booked *payment.Invoiced) { booked.FiatAmount = alpacadecimal.NewFromInt(6) },
			expectedError: payment.ErrPaymentAlreadyAuthorized,
		},
		{
			name: "deleted payment", mutate: func(booked *payment.Invoiced) { booked.DeletedAt = lo.ToPtr(booked.CreatedAt) },
			expectedError: payment.ErrPaymentAlreadyAuthorized,
		},
		{
			name: "missing authorization", mutate: func(booked *payment.Invoiced) { booked.Authorized = nil },
			invalidField: "authorization transaction data is missing",
		},
		{
			name: "empty authorization reference", mutate: func(booked *payment.Invoiced) { booked.Authorized.TransactionGroupID = "" },
			invalidField: "transaction group ID is required",
		},
		{
			name: "missing authorization time", mutate: func(booked *payment.Invoiced) { booked.Authorized.Time = time.Time{} },
			invalidField: "time is required",
		},
		{
			name: "invalid payment status", mutate: func(booked *payment.Invoiced) { booked.Status = "invalid" },
			invalidField: "invalid payment settlement status",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given an existing booking on a valid invoice-backed usage run
			in := BookInvoicedPaymentAuthorizedInput(newSettlePaymentInput(t))
			booked := in.Run.Payment
			require.NoError(t, in.Validate())
			if tc.mutate != nil {
				tc.mutate(booked)
			}
			if tc.invoiceAmount != nil {
				in.Line.Totals.Total = *tc.invoiceAmount
			}
			handler := &authorizationReplayHandler{}
			svc := &Service{handler: handler}

			// when authorization is replayed
			result, err := svc.BookInvoicedPaymentAuthorized(t.Context(), in)

			// then matching bookings retain their references without another journal call
			switch {
			case tc.expectedError != nil:
				require.ErrorIs(t, err, tc.expectedError)
			case tc.invalidField != "":
				require.ErrorContains(t, err, tc.invalidField)
			default:
				require.NoError(t, err)
				require.Equal(t, in.Run, result.Run)
				require.Same(t, booked, result.Payment)
			}
			require.False(t, handler.called)
		})
	}
}

type authorizationReplayHandler struct {
	usagebased.Handler
	called bool
}

func (h *authorizationReplayHandler) OnPaymentAuthorized(context.Context, usagebased.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
	h.called = true
	return ledgertransaction.GroupReference{}, errors.New("unexpected authorization booking")
}

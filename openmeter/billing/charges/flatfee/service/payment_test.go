package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestPostInvoicePaymentAuthorizedRecognizesMatchingBooking(t *testing.T) {
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
			// given a persisted booking that may conflict with the authorization being retried
			at := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
			period := timeutil.ClosedPeriod{From: at, To: at.AddDate(0, 1, 0)}
			booked := payment.Invoiced{
				Payment: payment.Payment{
					NamespacedID: models.NamespacedID{Namespace: "namespace", ID: "payment"},
					ManagedModel: models.ManagedModel{CreatedAt: at, UpdatedAt: at},
					Base: payment.Base{
						ServicePeriod: period,
						FiatAmount:    alpacadecimal.NewFromInt(5),
						Status:        payment.StatusAuthorized,
						Authorized: &ledgertransaction.TimedGroupReference{
							GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "authorization-group"},
							Time:           at,
						},
					},
				},
				InvoiceID: "invoice",
				LineID:    "line",
			}
			require.NoError(t, booked.Validate())
			if tc.mutate != nil {
				tc.mutate(&booked)
			}

			charge := flatfee.Charge{
				ChargeBase: flatfee.ChargeBase{
					ManagedResource: meta.ManagedResource{NamespacedModel: models.NamespacedModel{Namespace: "namespace"}, ID: "charge"},
				},
				Realizations: flatfee.Realizations{CurrentRun: &flatfee.RealizationRun{
					RealizationRunBase: flatfee.RealizationRunBase{
						ID:            flatfee.RealizationRunID{Namespace: "namespace", ID: "run"},
						LineID:        lo.ToPtr("line"),
						InvoiceID:     lo.ToPtr("invoice"),
						ServicePeriod: period,
					},
					Payment: &booked,
				}},
			}
			line := &billing.StandardLine{
				StandardLineBase: billing.StandardLineBase{
					ManagedResource: models.ManagedResource{NamespacedModel: models.NamespacedModel{Namespace: "namespace"}, ID: "line", Name: "flat fee"},
					InvoiceID:       "invoice",
					ChargeID:        lo.ToPtr("charge"),
					Currency:        currencyx.FiatCode("USD"),
					ManagedBy:       billing.SystemManagedLine,
					Period:          period,
					Totals:          totals.Totals{Total: alpacadecimal.NewFromInt(5)},
				},
				UsageBased: &billing.UsageBasedLine{Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount: alpacadecimal.NewFromInt(5), PaymentTerm: productcatalog.InAdvancePaymentTerm,
				})},
			}
			if tc.invoiceAmount != nil {
				line.Totals.Total = *tc.invoiceAmount
			}

			input := billing.StandardLineWithInvoiceHeader{Line: line, Invoice: billing.StandardInvoice{
				StandardInvoiceBase: billing.StandardInvoiceBase{Namespace: "namespace", ID: "invoice"},
			}}
			require.NoError(t, input.Validate())
			ctx, err := transaction.SetDriverOnContext(t.Context(), flatFeeBillabilityTransaction{})
			require.NoError(t, err)
			handler := &authorizationReplayHandler{}
			svc := &service{handler: handler}

			// when the authorization handler receives the same line again
			err = svc.postInvoicePaymentAuthorized(ctx, charge, input)

			// then only matching valid bookings succeed and the journal is never invoked again
			switch {
			case tc.expectedError != nil:
				require.ErrorIs(t, err, tc.expectedError)
			case tc.invalidField != "":
				require.ErrorContains(t, err, tc.invalidField)
			default:
				require.NoError(t, err)
			}

			require.False(t, handler.called)
		})
	}
}

type authorizationReplayHandler struct {
	flatfee.Handler
	called bool
}

func (h *authorizationReplayHandler) OnPaymentAuthorized(context.Context, flatfee.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
	h.called = true
	return ledgertransaction.GroupReference{}, errors.New("unexpected authorization booking")
}

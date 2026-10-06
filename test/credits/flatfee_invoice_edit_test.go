package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *CreditThenInvoiceTestSuite) TestFlatFeeInvoiceDiscountEditDoesNotProrateTwice() {
	// given: a subscription-managed flat fee covers half of its full service period
	// when: an API-originated draft edit changes only the percentage discount
	result := s.createAndDiscountFlatFeeDraft(timeutil.ClosedPeriod{
		From: time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	// then: the override owns the displayed gross without prorating it again
	s.RequireTotals(billingtest.ExpectedTotals{Amount: 50, Total: 50}, result.Before.Totals)
	s.Equal(result.Before.Lines.OrEmpty()[0].Period, result.After.Lines.OrEmpty()[0].Period)
	s.Equal(float64(50), result.After.Totals.Amount.InexactFloat64(), "a discount-only edit must not prorate the displayed price again")
	s.RequireTotals(billingtest.ExpectedTotals{Amount: 50, DiscountsTotal: 25, Total: 25}, result.After.Totals)
	s.Require().NotNil(result.Charge.Intent.GetOverrideLayerMutableFields())
	s.Equal(float64(50), result.Charge.Intent.GetEffectiveIntent().AmountBeforeProration.InexactFloat64())
	s.False(result.Charge.Intent.GetEffectiveIntent().ProRating.Enabled)
	s.Equal(float64(100), result.Charge.Intent.GetBaseIntent().AmountBeforeProration.InexactFloat64())
	s.True(result.Charge.Intent.GetBaseIntent().ProRating.Enabled)
}

func (s *CreditThenInvoiceTestSuite) TestFlatFeeInvoiceManualEditsKeepAbsoluteAmount() {
	// given: a prorated subscription charge has a manually discounted draft line
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	result := s.createAndDiscountFlatFeeDraft(servicePeriod)
	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	s.Require().NotNil(result.Charge.Realizations.CurrentRun)
	base := result.Charge.Intent.GetBaseIntent()
	lineID := result.After.Lines.OrEmpty()[0].ID
	shortPeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC),
	}

	for _, test := range []struct {
		name           string
		edit           func(*billing.StandardInvoice) error
		expectedTotals billingtest.ExpectedTotals
		expectedPeriod timeutil.ClosedPeriod
	}{
		{
			name: "name-only edit preserves gross",
			edit: func(invoice *billing.StandardInvoice) error {
				invoice.Lines.GetByID(lineID).Name = "manually edited flat fee"
				return nil
			},
			expectedTotals: billingtest.ExpectedTotals{Amount: 50, DiscountsTotal: 25, Total: 25},
			expectedPeriod: servicePeriod,
		},
		{
			name: "explicit price is the gross for half service",
			edit: func(invoice *billing.StandardInvoice) error {
				invoice.Lines.GetByID(lineID).SetPrice(*productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(80),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}))
				return nil
			},
			expectedTotals: billingtest.ExpectedTotals{Amount: 80, DiscountsTotal: 40, Total: 40},
			expectedPeriod: servicePeriod,
		},
		{
			name: "shortening service does not scale the manual gross",
			edit: func(invoice *billing.StandardInvoice) error {
				invoice.Lines.GetByID(lineID).Period = shortPeriod
				return nil
			},
			expectedTotals: billingtest.ExpectedTotals{Amount: 80, DiscountsTotal: 40, Total: 40},
			expectedPeriod: shortPeriod,
		},
		{
			name: "another discount edit keeps the gross",
			edit: func(invoice *billing.StandardInvoice) error {
				invoice.Lines.GetByID(lineID).RateCardDiscounts.Percentage.Percentage = models.NewPercentage(20)
				return nil
			},
			expectedTotals: billingtest.ExpectedTotals{Amount: 80, DiscountsTotal: 16, Total: 64},
			expectedPeriod: shortPeriod,
		},
		{
			name: "extending service does not scale the manual gross",
			edit: func(invoice *billing.StandardInvoice) error {
				invoice.Lines.GetByID(lineID).Period = servicePeriod
				return nil
			},
			expectedTotals: billingtest.ExpectedTotals{Amount: 80, DiscountsTotal: 16, Total: 64},
			expectedPeriod: servicePeriod,
		},
	} {
		s.Run(test.name, func() {
			// given: the previous edit's persisted manual override is effective
			// when: the same draft line is edited again through the API
			invoice, err := s.BillingService.UpdateStandardInvoice(s.T().Context(), billing.UpdateStandardInvoiceInput{
				Invoice:      result.After.GetInvoiceID(),
				ChangeSource: billing.ChangeSourceAPIRequest,
				EditFn:       test.edit,
			})
			s.Require().NoError(err)

			// then: the same line and run use the absolute gross while the source intent stays unchanged
			s.Require().Len(invoice.Lines.OrEmpty(), 1)
			line := invoice.Lines.GetByID(lineID)
			s.Require().NotNil(line)
			s.Equal(billing.ManuallyManagedLine, line.ManagedBy)
			s.Equal(test.expectedPeriod, line.Period)
			s.RequireTotals(test.expectedTotals, invoice.Totals)
			charge := s.mustGetFlatFeeChargeByIDWithExpands(result.Charge.GetChargeID(), meta.Expands{meta.ExpandRealizations})
			s.True(base.Equal(charge.Intent.GetBaseIntent()), "manual edits must preserve the subscription source intent")
			s.False(charge.Intent.GetEffectiveIntent().ProRating.Enabled)
			s.Equal(test.expectedTotals.Amount, charge.Intent.GetEffectiveIntent().AmountBeforeProration.InexactFloat64())
			s.Equal(test.expectedPeriod, charge.Intent.GetEffectiveServicePeriod())
			s.Require().NotNil(charge.Realizations.CurrentRun)
			s.Equal(result.Charge.Realizations.CurrentRun.ID, charge.Realizations.CurrentRun.ID)
			s.Equal(test.expectedTotals.Amount, charge.Realizations.CurrentRun.AmountAfterProration.InexactFloat64())
		})
	}

	s.Run("clearing the override restores subscription proration", func() {
		// given: the draft line's override owns an absolute gross of 80
		// when: the customer clears that override
		_, err := s.Charges.ClearCustomerChargeOverride(s.T().Context(), charges.ClearCustomerChargeOverrideInput{
			Namespace:  result.Charge.Namespace,
			CustomerID: base.CustomerID,
			ChargeID:   result.Charge.ID,
		})
		s.Require().NoError(err)

		// then: the effective amount returns to 100 prorated for half service
		charge := s.mustGetFlatFeeChargeByIDWithExpands(result.Charge.GetChargeID(), meta.Expands{meta.ExpandRealizations})
		s.Nil(charge.Intent.GetOverrideLayerMutableFields())
		s.True(charge.Intent.GetEffectiveIntent().ProRating.Enabled)
		s.Equal(float64(100), charge.Intent.GetEffectiveIntent().AmountBeforeProration.InexactFloat64())
		s.Equal(float64(50), charge.State.AmountAfterProration.InexactFloat64())
		s.Equal(servicePeriod, charge.Intent.GetEffectiveServicePeriod())
	})
}

type flatFeeDiscountEditResult struct {
	Before billing.StandardInvoice
	After  billing.StandardInvoice
	Charge flatfee.Charge
}

// createAndDiscountFlatFeeDraft exercises the charge-to-invoice boundary and the
// API edit ownership transition without involving an external payment provider.
func (s *CreditThenInvoiceTestSuite) createAndDiscountFlatFeeDraft(servicePeriod timeutil.ClosedPeriod) flatFeeDiscountEditResult {
	s.T().Helper()
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("flatfee-invoice-discount-edit")
	s.ProvisionDefaultTaxCodes(ctx, ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(), billingtest.WithManualApproval())

	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	intent, err := s.CreateMockChargeIntent(CreateMockChargeIntentInput{
		Customer:      cust.GetID(),
		Currency:      USD,
		ServicePeriod: servicePeriod,
		Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(100),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		}),
		Name:              "flat fee",
		SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
		ManagedBy:         billing.SubscriptionManagedLine,
		UniqueReferenceID: "flatfee-invoice-discount-edit",
		ProRating: productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		},
	}).AsFlatFeeIntent()
	s.Require().NoError(err)
	intent.FullServicePeriod = timeutil.ClosedPeriod{
		From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	intent.BillingPeriod = intent.FullServicePeriod
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents:   charges.NewCreateChargeIntents(charges.NewChargeIntent(intent)),
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	charge, err := created[0].AsFlatFeeCharge()
	s.Require().NoError(err)

	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.From),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	before := invoices[0]
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, before.Status)
	s.Require().Len(before.Lines.OrEmpty(), 1)
	line := before.Lines.OrEmpty()[0]
	s.Equal(billing.LineEngineTypeChargeFlatFee, line.Engine)
	s.Equal(billing.SubscriptionManagedLine, line.ManagedBy)
	s.Require().Len(line.DetailedLines, 1)
	s.Empty(line.DetailedLines[0].AmountDiscounts)

	_, err = s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      before.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.StandardInvoice) error {
			invoice.Lines.GetByID(line.ID).RateCardDiscounts = billing.Discounts{
				Percentage: &billing.PercentageDiscount{
					PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
					CorrelationID:      "draft-edit-percentage",
				},
			}
			return nil
		},
	})
	s.Require().NoError(err)
	after, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: before.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpands{billing.StandardInvoiceExpandLines},
	})
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, after.Status)
	s.Require().Len(after.Lines.OrEmpty(), 1)
	s.Equal(line.ID, after.Lines.OrEmpty()[0].ID)
	s.Equal(billing.ManuallyManagedLine, after.Lines.OrEmpty()[0].ManagedBy)

	return flatFeeDiscountEditResult{
		Before: before,
		After:  after,
		Charge: s.mustGetFlatFeeChargeByIDWithExpands(charge.GetChargeID(), meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDetailedLines,
		}),
	}
}

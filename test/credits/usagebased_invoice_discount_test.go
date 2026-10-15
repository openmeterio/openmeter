package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargedetailedline "github.com/openmeterio/openmeter/openmeter/billing/charges/models/detailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *CreditThenInvoiceTestSuite) TestUsageBasedAmountDiscountsBillProgressively() {
	for _, test := range []struct {
		name                    string
		price                   *productcatalog.Price
		discounts               billing.Discounts
		usageBeforeFirstInvoice float64
		usageSent               float64
		expectedTotals          billingtest.ExpectedTotals
		expectedAmountDiscounts chargedetailedline.AmountDiscounts
	}{
		{
			name: "100 units at USD 1 per unit with a 50% discount",
			price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
			discounts: billing.Discounts{
				Percentage: &billing.PercentageDiscount{
					PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
					CorrelationID:      "usage-percentage-discount",
				},
			},
			usageSent: 100,
			expectedTotals: billingtest.ExpectedTotals{
				Amount:         100,
				DiscountsTotal: 50,
				Total:          50,
			},
			expectedAmountDiscounts: chargedetailedline.AmountDiscounts{
				{
					ChildUniqueReferenceID: "rateCardDiscount/correlationID=usage-percentage-discount",
					Reason: billing.NewDiscountReasonFrom(billing.PercentageDiscount{
						PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
						CorrelationID:      "usage-percentage-discount",
					}),
					Amount: alpacadecimal.NewFromInt(50),
				},
			},
		},
		{
			name: "40 units at USD 2 per unit with a USD 60 maximum spend",
			price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount:        alpacadecimal.NewFromInt(2),
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(60)),
			}),
			usageSent: 40,
			expectedTotals: billingtest.ExpectedTotals{
				Amount:         80,
				DiscountsTotal: 20,
				Total:          60,
			},
			expectedAmountDiscounts: chargedetailedline.AmountDiscounts{
				{
					ChildUniqueReferenceID: billing.LineMaximumSpendReferenceID,
					Description:            lo.ToPtr("Maximum spend discount for charges over 60"),
					Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
					Amount:                 alpacadecimal.NewFromInt(20),
				},
			},
		},
		{
			name: "40 units already billed at the USD 60 maximum spend, then 40 more units",
			price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount:        alpacadecimal.NewFromInt(2),
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(60)),
			}),
			usageBeforeFirstInvoice: 40,
			usageSent:               40,
			expectedTotals: billingtest.ExpectedTotals{
				Amount:         80,
				DiscountsTotal: 80,
				Total:          0,
			},
			expectedAmountDiscounts: chargedetailedline.AmountDiscounts{
				{
					ChildUniqueReferenceID: billing.LineMaximumSpendReferenceID,
					Description:            lo.ToPtr("Maximum spend discount for charges over 60"),
					Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
					Amount:                 alpacadecimal.NewFromInt(80),
				},
			},
		},
	} {
		s.Run(test.name, func() {
			// given: a discounted charge with progressive billing enabled
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("usagebased-invoice-discount")
			s.ProvisionDefaultTaxCodes(ctx, ns)
			customInvoicing := s.SetupCustomInvoicing(ns)
			cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
			s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(), billingtest.WithManualApproval(), billingtest.WithProgressiveBilling())
			feature := s.SetupApiRequestsTotalFeature(ctx, ns)
			defer feature.Cleanup()
			period := timeutil.ClosedPeriod{
				From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			}
			clock.FreezeTime(period.From.Add(-time.Hour))
			defer clock.UnFreeze()
			if test.usageBeforeFirstInvoice > 0 {
				s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, test.usageBeforeFirstInvoice, period.From.Add(time.Hour))
			}

			intent, err := s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          cust.GetID(),
				Currency:          USD,
				ServicePeriod:     period,
				Price:             test.price,
				Name:              "usage",
				SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usagebased-invoice-discount",
				FeatureKey:        feature.Feature.Key,
			}).AsUsageBasedIntent()
			s.Require().NoError(err)
			intent.Discounts = test.discounts
			created, err := s.Charges.Create(ctx, charges.CreateInput{
				Namespace: ns,
				Intents:   charges.NewCreateChargeIntents(charges.NewChargeIntent(intent)),
			})
			s.Require().NoError(err)
			s.Require().Len(created, 1)
			charge, err := created[0].AsUsageBasedCharge()
			s.Require().NoError(err)

			// when: billing requests a progressive invoice during the service period
			midPeriod := period.From.Add(15 * 24 * time.Hour)
			clock.FreezeTime(midPeriod)
			defer clock.UnFreeze()
			invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: cust.GetID(),
				AsOf:     lo.ToPtr(midPeriod),
			})

			// then: the charge will be invoiced, creating a run mid-period
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)
			partiallyBilledCharge := s.mustGetUsageBasedChargeByIDWithExpands(charge.GetChargeID(), meta.Expands{
				meta.ExpandRealizations,
			})
			s.Len(partiallyBilledCharge.Realizations, 1)
			s.NotEmpty(partiallyBilledCharge.State.CurrentRealizationRunID)

			clock.FreezeTime(invoices[0].DefaultCollectionAtForStandardInvoice())
			defer clock.UnFreeze()
			_, err = s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
			s.Require().NoError(err)
			_, err = s.BillingService.ApproveInvoice(ctx, invoices[0].GetInvoiceID())
			s.Require().NoError(err)

			// when: usage arrives after the first invoice and billing invoices the rest of the period at its end
			s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, test.usageSent, midPeriod.Add(time.Hour))
			clock.FreezeTime(period.To.Add(time.Second))
			defer clock.UnFreeze()
			invoices, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: cust.GetID(),
				AsOf:     lo.ToPtr(period.To),
			})
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)
			invoice := invoices[0]
			charge = s.mustGetUsageBasedChargeByIDWithExpands(charge.GetChargeID(), meta.Expands{
				meta.ExpandRealizations,
				meta.ExpandDetailedLines,
			})
			run, err := charge.GetCurrentRealizationRun()
			s.Require().NoError(err)
			runDetails := run.DetailedLines.OrEmpty()
			s.Require().Len(runDetails, 1)
			s.Require().Len(runDetails[0].AmountDiscounts, len(test.expectedAmountDiscounts))
			for idx, expectedDiscount := range test.expectedAmountDiscounts {
				actualDiscount := runDetails[0].AmountDiscounts[idx]
				s.Equal(expectedDiscount.ChildUniqueReferenceID, actualDiscount.ChildUniqueReferenceID)
				s.Equal(expectedDiscount.Description, actualDiscount.Description)
				s.Equal(expectedDiscount.Reason, actualDiscount.Reason)
				s.Equal(expectedDiscount.Amount.InexactFloat64(), actualDiscount.Amount.InexactFloat64())
				s.Equal(expectedDiscount.RoundingAmount.InexactFloat64(), actualDiscount.RoundingAmount.InexactFloat64())
			}

			// then: the persisted invoice retains the same facts as a billing-managed discount
			s.Require().Len(invoice.Lines.OrEmpty(), 1)
			line := invoice.Lines.OrEmpty()[0]
			s.Equal(timeutil.ClosedPeriod{
				From: midPeriod,
				To:   period.To,
			}, line.Period)
			s.Equal(billing.LineEngineTypeChargeUsageBased, line.Engine)
			s.RequireTotals(test.expectedTotals, line.Totals)
			s.Require().Len(line.DetailedLines, 1)
			detail := line.DetailedLines[0]
			s.Require().Len(detail.AmountDiscounts, len(test.expectedAmountDiscounts), "invoice mapping must preserve the charge's discount breakdown")
			s.Require().NoError(detail.Validate())
			s.NotEmpty(detail.ChildUniqueReferenceID)
			discount := detail.AmountDiscounts[0]
			s.NotEmpty(discount.ID)
			s.Equal(runDetails[0].AmountDiscounts[0].ChildUniqueReferenceID, lo.FromPtr(discount.ChildUniqueReferenceID))
			s.Equal(runDetails[0].AmountDiscounts[0].Reason, discount.Reason)
			s.Equal(test.expectedAmountDiscounts[0].Amount.InexactFloat64(), discount.Amount.InexactFloat64())

			// when: collection completes and maps the same realization into the invoice again
			clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
			defer clock.UnFreeze()
			_, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
			s.Require().NoError(err)
			updated, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: invoice.GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpands{billing.StandardInvoiceExpandLines},
			})
			s.Require().NoError(err)

			// then: billing retains the detailed-line and discount IDs across collection
			s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, updated.Status)
			s.RequireTotals(test.expectedTotals, updated.Totals)
			s.Require().Len(updated.Lines.OrEmpty(), 1)
			updatedLine := updated.Lines.OrEmpty()[0]
			s.Equal(line.ID, updatedLine.ID)
			s.Require().Len(updatedLine.DetailedLines, 1)
			updatedDetail := updatedLine.DetailedLines[0]
			s.Require().NoError(updatedDetail.Validate())
			s.Equal(detail.ID, updatedDetail.ID)
			s.Equal(detail.ChildUniqueReferenceID, updatedDetail.ChildUniqueReferenceID)
			s.Require().Len(updatedDetail.AmountDiscounts, len(test.expectedAmountDiscounts))
			updatedDiscount := updatedDetail.AmountDiscounts[0]
			s.Equal(discount.ID, updatedDiscount.ID)
			s.Equal(discount.CreatedAt, updatedDiscount.CreatedAt)
			s.Equal(discount.ChildUniqueReferenceID, updatedDiscount.ChildUniqueReferenceID)
			s.Nil(updatedDiscount.DeletedAt)
			s.Equal(test.expectedAmountDiscounts[0].Amount.InexactFloat64(), updatedDiscount.Amount.InexactFloat64())
			s.Equal(usagebased.RealizationRunTypeFinalRealization, run.Type)
		})
	}
}

func (s *CreditThenInvoiceTestSuite) TestPercentageDiscountProgressiveBilling() {
	// Given:
	// - Service period: 2026-01-01 … 2026-01-30
	// - A usage based charge with a graduated tiered price at two tiers (10 units @ 5, then @ 4)
	// - A 25% percentage discount on the charge
	// - Invoice 1 exists for service period 2026-01-01 … 2026-01-15
	// - Metered usage exists for the whole period

	// When
	// - Invoice 2 is created at the end of the period
	// Then:
	// - The invoice 2 has properly populated discounts
	// - The discount only captures 25% of the actually billed amount
	// - Totals captures the right discount amount
	// - AmountDiscounts capture the discount amounts and matches the detailed line totals
	periodStart := lo.Must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2026-01-30T00:00:00Z"))
	clock.FreezeTime(periodStart)
	defer clock.UnFreeze()
	namespace := "ns-percentage-discounts-usage-progressive"
	ctx := s.T().Context()

	customInvoicing := s.SetupCustomInvoicing(namespace)

	s.ProvisionDefaultTaxCodes(ctx, namespace)
	s.ProvisionBillingProfile(ctx, namespace, customInvoicing.App.GetID(), billingtest.WithManualApproval(), billingtest.WithProgressiveBilling())

	customerEntity := s.CreateLedgerBackedCustomer(namespace, "test-customer")
	s.NotNil(customerEntity)

	meterSlug := "flat-per-unit"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ID:            meterID,
			Namespace:     namespace,
			CreatedAt:     clock.Now(),
			UpdatedAt:     clock.Now(),
			Name:          "Flat per unit",
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))

	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 8, periodStart.Add(time.Minute))

	defer func() {
		err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
		s.Require().NoError(err, "meter adapter replace meters")
	}()

	featureFlatPerUnit := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterID:   lo.ToPtr(meterID),
	}))

	res, err := s.Charges.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.FiatCode(currency.USD),
			Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
				{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: namespace,
						Name:      "Test item1",
					}),
					ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},

					InvoiceAt: periodEnd,

					ManagedBy: billing.ManuallyManagedLine,

					Currency: currencyx.FiatCode(currency.USD),
					RateCardDiscounts: billing.Discounts{
						Percentage: &billing.PercentageDiscount{
							Percentage: models.NewPercentage(25),
						},
					},
					FeatureKey: featureFlatPerUnit.Key,
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.TieredPrice{
						Mode: productcatalog.GraduatedTieredPrice,
						Tiers: []productcatalog.PriceTier{
							{
								UnitPrice: &productcatalog.PriceTierUnitPrice{
									Amount: alpacadecimal.NewFromFloat(5),
								},
								UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
							},
							{
								UnitPrice: &productcatalog.PriceTierUnitPrice{
									Amount: alpacadecimal.NewFromFloat(4),
								},
							},
						},
					})),
				},
			}),
		})
	s.Require().NoError(err)
	s.Require().Len(res.Lines, 1)

	s.Empty(res.Lines[0].RateCardDiscounts.Usage)

	invoice1AsOf := lo.Must(time.Parse(time.RFC3339, "2026-01-15T00:00:00Z"))
	clock.FreezeTime(invoice1AsOf)
	defer clock.UnFreeze()

	// Invoice 1 bills the first 8 units, all within tier 1
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
		AsOf:     lo.ToPtr(invoice1AsOf),
	})
	s.NoError(err)
	s.Len(invoices, 1)
	s.Len(invoices[0].Lines.OrEmpty(), 1)

	invoiceLine := invoices[0].Lines.OrEmpty()[0]
	s.Require().Len(invoiceLine.DetailedLines, 1)
	detailedLine := invoiceLine.DetailedLines[0]
	s.Equal(float64(8), detailedLine.Quantity.InexactFloat64())
	s.Require().Len(detailedLine.AmountDiscounts, 1)
	s.Equal(billing.RatecardPercentageDiscountReason, detailedLine.AmountDiscounts[0].Reason.Type())

	// The line has 8 quantity, 8 metered quantity
	s.Equal(float64(8), invoiceLine.UsageBased.Quantity.InexactFloat64())
	s.Equal(float64(8), invoiceLine.UsageBased.MeteredQuantity.InexactFloat64())

	s.Equal(float64(0), invoiceLine.UsageBased.PreLinePeriodQuantity.InexactFloat64())
	s.Equal(float64(0), invoiceLine.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())

	s.Equal(float64(40), invoiceLine.Totals.Amount.InexactFloat64())
	s.Equal(float64(40), detailedLine.Totals.Amount.InexactFloat64())
	s.Equal(float64(40), invoices[0].Totals.Amount.InexactFloat64())

	s.Equal(float64(10), invoiceLine.Totals.DiscountsTotal.InexactFloat64())
	s.Equal(float64(10), detailedLine.AmountDiscounts[0].Amount.InexactFloat64())
	s.Equal(float64(0), detailedLine.AmountDiscounts[0].RoundingAmount.InexactFloat64(), "RoundingAmount")
	s.Equal(float64(10), detailedLine.Totals.DiscountsTotal.InexactFloat64())
	s.Equal(float64(10), invoices[0].Totals.DiscountsTotal.InexactFloat64())

	s.Equal(float64(30), invoiceLine.Totals.Total.InexactFloat64())
	s.Equal(float64(30), detailedLine.Totals.Total.InexactFloat64())
	s.Equal(float64(30), invoices[0].Totals.Total.InexactFloat64())

	clock.FreezeTime(invoices[0].DefaultCollectionAtForStandardInvoice())
	defer clock.UnFreeze()
	_, err = s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
	s.Require().NoError(err)
	_, err = s.BillingService.ApproveInvoice(ctx, invoices[0].GetInvoiceID())
	s.Require().NoError(err)

	s.Run("[invoice2] Creating a draft invoice with 4 more usage", func() {
		invoice2AsOf := lo.Must(time.Parse(time.RFC3339, "2026-01-30T00:00:00Z"))
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, 4, invoice2AsOf.Add(-30*time.Hour))
		clock.FreezeTime(invoice2AsOf)
		defer clock.UnFreeze()

		// Invoice 2 bills only the delta since invoice 1
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     lo.ToPtr(invoice2AsOf),
		})
		s.NoError(err)
		s.Require().Len(invoices, 1)
		s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

		invoiceLine := invoices[0].Lines.OrEmpty()[0]
		s.Require().Nil(invoiceLine.SplitLineHierarchy)

		// The line has 4 quantity, 4 metered quantity
		s.Equal(float64(4), invoiceLine.UsageBased.Quantity.InexactFloat64())
		s.Equal(float64(4), invoiceLine.UsageBased.MeteredQuantity.InexactFloat64())

		s.Equal(float64(8), invoiceLine.UsageBased.PreLinePeriodQuantity.InexactFloat64())
		s.Equal(float64(8), invoiceLine.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())

		// The 4 new units split across both tiers: 2 complete tier 1, 2 open tier 2
		s.Require().Len(invoiceLine.DetailedLines, 2)

		s.Equal(float64(18), invoiceLine.Totals.Amount.InexactFloat64())
		s.Equal(float64(4.5), invoiceLine.Totals.DiscountsTotal.InexactFloat64())
		s.Equal(float64(13.5), invoiceLine.Totals.Total.InexactFloat64())
		s.Equal(float64(18), invoices[0].Totals.Amount.InexactFloat64())
		s.Equal(float64(4.5), invoices[0].Totals.DiscountsTotal.InexactFloat64())
		s.Equal(float64(13.5), invoices[0].Totals.Total.InexactFloat64())

		detailedLine1 := invoiceLine.DetailedLines[0]
		s.Equal(float64(2), detailedLine1.Quantity.InexactFloat64())
		s.Require().Len(detailedLine1.AmountDiscounts, 1)
		s.Equal(billing.RatecardPercentageDiscountReason, detailedLine1.AmountDiscounts[0].Reason.Type())

		s.Equal(float64(10), detailedLine1.Totals.Amount.InexactFloat64())
		s.Equal(float64(2.5), detailedLine1.AmountDiscounts[0].Amount.InexactFloat64())
		s.Equal(float64(0), detailedLine1.AmountDiscounts[0].RoundingAmount.InexactFloat64())
		s.Equal(float64(7.5), detailedLine1.Totals.Total.InexactFloat64())

		detailedLine2 := invoiceLine.DetailedLines[1]
		s.Equal(float64(2), detailedLine2.Quantity.InexactFloat64())
		s.Require().Len(detailedLine2.AmountDiscounts, 1)
		s.Equal(billing.RatecardPercentageDiscountReason, detailedLine2.AmountDiscounts[0].Reason.Type())

		s.Equal(float64(8), detailedLine2.Totals.Amount.InexactFloat64())
		s.Equal(float64(2), detailedLine2.AmountDiscounts[0].Amount.InexactFloat64())
		s.Equal(float64(0), detailedLine2.AmountDiscounts[0].RoundingAmount.InexactFloat64())
		s.Equal(float64(6), detailedLine2.Totals.Total.InexactFloat64())

		s.Equal(detailedLine1.AmountDiscounts[0].ChildUniqueReferenceID, detailedLine2.AmountDiscounts[0].ChildUniqueReferenceID)
	})
}

func (s *CreditThenInvoiceTestSuite) TestPercentageDiscountGraduatedTiersSingleInvoice() {
	// Given:
	// - Service period: 2026-01-01 … 2026-01-30
	// - An usage based charge with a graduated tiered price at two tiers (10 units @ 5, then @ 4)
	// - A 25% percentage discount on the charge
	// - Metered usage of 12 units, crossing the first tier boundary

	// When
	// - A single invoice is created for the whole service period
	// Then:
	// - The line has two detailed lines, one per tier
	// - Each detailed line carries its own 25% amount discount
	// - Line and invoice totals equal the sum of the detailed line totals
	periodStart := lo.Must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2026-01-30T00:00:00Z"))
	clock.FreezeTime(periodStart)
	defer clock.UnFreeze()
	namespace := "ns-percentage-discounts-graduated-single-invoice"
	ctx := s.T().Context()

	defer s.MockStreamingConnector.Reset()

	customInvoicing := s.SetupCustomInvoicing(namespace)

	s.ProvisionDefaultTaxCodes(ctx, namespace)
	s.ProvisionBillingProfile(ctx, namespace, customInvoicing.App.GetID(), billingtest.WithManualApproval(), billingtest.WithProgressiveBilling())

	customerEntity := s.CreateLedgerBackedCustomer(namespace, "test-customer")
	s.NotNil(customerEntity)

	meterSlug := "flat-per-unit"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ID:            meterID,
			Namespace:     namespace,
			CreatedAt:     clock.Now(),
			UpdatedAt:     clock.Now(),
			Name:          "Flat per unit",
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))

	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 8, periodStart.Add(time.Minute))
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 4, periodEnd.Add(-30*time.Hour))

	defer func() {
		err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
		s.Require().NoError(err, "meter adapter replace meters")
	}()

	featureFlatPerUnit := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterID:   lo.ToPtr(meterID),
	}))

	res, err := s.Charges.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.FiatCode(currency.USD),
			Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
				{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: namespace,
						Name:      "Test item1",
					}),
					ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},

					InvoiceAt: periodEnd,

					ManagedBy: billing.ManuallyManagedLine,

					Currency: currencyx.FiatCode(currency.USD),
					RateCardDiscounts: billing.Discounts{
						Percentage: &billing.PercentageDiscount{
							Percentage: models.NewPercentage(25),
						},
					},
					FeatureKey: featureFlatPerUnit.Key,
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.TieredPrice{
						Mode: productcatalog.GraduatedTieredPrice,
						Tiers: []productcatalog.PriceTier{
							{
								UnitPrice: &productcatalog.PriceTierUnitPrice{
									Amount: alpacadecimal.NewFromFloat(5),
								},
								UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
							},
							{
								UnitPrice: &productcatalog.PriceTierUnitPrice{
									Amount: alpacadecimal.NewFromFloat(4),
								},
							},
						},
					})),
				},
			}),
		})
	s.Require().NoError(err)
	s.Require().Len(res.Lines, 1)

	clock.FreezeTime(periodEnd)
	defer clock.UnFreeze()

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
		AsOf:     lo.ToPtr(periodEnd),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Empty(invoices[0].ValidationIssues)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

	invoiceLine := invoices[0].Lines.OrEmpty()[0]

	// The line has 12 quantity, 12 metered quantity
	s.Equal(float64(12), invoiceLine.UsageBased.Quantity.InexactFloat64())
	s.Equal(float64(12), invoiceLine.UsageBased.MeteredQuantity.InexactFloat64())

	s.Equal(float64(0), invoiceLine.UsageBased.PreLinePeriodQuantity.InexactFloat64())
	s.Equal(float64(0), invoiceLine.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())

	// One detailed line per tier
	s.Require().Len(invoiceLine.DetailedLines, 2)

	s.Equal(float64(58), invoiceLine.Totals.Amount.InexactFloat64())
	s.Equal(float64(14.5), invoiceLine.Totals.DiscountsTotal.InexactFloat64())
	s.Equal(float64(43.5), invoiceLine.Totals.Total.InexactFloat64())
	s.Equal(float64(58), invoices[0].Totals.Amount.InexactFloat64())
	s.Equal(float64(14.5), invoices[0].Totals.DiscountsTotal.InexactFloat64())
	s.Equal(float64(43.5), invoices[0].Totals.Total.InexactFloat64())

	detailedLine1 := invoiceLine.DetailedLines[0]
	s.Equal(float64(10), detailedLine1.Quantity.InexactFloat64())
	s.Equal(float64(5), detailedLine1.PerUnitAmount.InexactFloat64())
	s.Require().Len(detailedLine1.AmountDiscounts, 1)
	s.Equal(billing.RatecardPercentageDiscountReason, detailedLine1.AmountDiscounts[0].Reason.Type())

	s.Equal(float64(50), detailedLine1.Totals.Amount.InexactFloat64())
	s.Equal(float64(12.5), detailedLine1.AmountDiscounts[0].Amount.InexactFloat64())
	s.Equal(float64(0), detailedLine1.AmountDiscounts[0].RoundingAmount.InexactFloat64())
	s.Equal(float64(12.5), detailedLine1.Totals.DiscountsTotal.InexactFloat64())
	s.Equal(float64(37.5), detailedLine1.Totals.Total.InexactFloat64())

	detailedLine2 := invoiceLine.DetailedLines[1]
	s.Equal(float64(2), detailedLine2.Quantity.InexactFloat64())
	s.Equal(float64(4), detailedLine2.PerUnitAmount.InexactFloat64())
	s.Require().Len(detailedLine2.AmountDiscounts, 1)
	s.Equal(billing.RatecardPercentageDiscountReason, detailedLine2.AmountDiscounts[0].Reason.Type())

	s.Equal(float64(8), detailedLine2.Totals.Amount.InexactFloat64())
	s.Equal(float64(2), detailedLine2.AmountDiscounts[0].Amount.InexactFloat64())
	s.Equal(float64(0), detailedLine2.AmountDiscounts[0].RoundingAmount.InexactFloat64())
	s.Equal(float64(2), detailedLine2.Totals.DiscountsTotal.InexactFloat64())
	s.Equal(float64(6), detailedLine2.Totals.Total.InexactFloat64())
}

func (s *CreditThenInvoiceTestSuite) TestPercentageDiscountProgressiveBillingUsageShrinkIsRejected() {
	// Given:
	// - Service period: 2026-01-01 … 2026-01-30
	// - An usage based charge with a graduated tiered price at two tiers (10 units @ 5, then @ 4)
	// - A 25% percentage discount on the charge
	// - 12 units of usage in the first half of the period
	// - Invoice 1 for 2026-01-01 … 2026-01-15 is approved with both tiers billed

	// When
	// - 4 units are retracted in the second half, so the cumulative usage drops to 8
	// - Invoice 2 is created at the end of the period
	// Then:
	// - The delta run would owe the customer money, which the delta rater does not support yet
	// - Invoice 2 is created in an invalid state with the negative total recorded as a critical issue
	// - The charge line carries no detailed lines, so no negative discount reaches the invoice
	//
	// Once corrections are supported, this scenario is expected to produce a tier 1 delta of
	// -2 units with a -2.5 discount and a full tier 2 reversal of -2 units with a -2 discount.
	periodStart := lo.Must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2026-01-30T00:00:00Z"))
	clock.FreezeTime(periodStart)
	defer clock.UnFreeze()
	namespace := "ns-percentage-discounts-progressive-usage-shrink"
	ctx := s.T().Context()

	defer s.MockStreamingConnector.Reset()

	customInvoicing := s.SetupCustomInvoicing(namespace)

	s.ProvisionDefaultTaxCodes(ctx, namespace)
	s.ProvisionBillingProfile(ctx, namespace, customInvoicing.App.GetID(), billingtest.WithManualApproval(), billingtest.WithProgressiveBilling())

	customerEntity := s.CreateLedgerBackedCustomer(namespace, "test-customer")
	s.NotNil(customerEntity)

	meterSlug := "flat-per-unit"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ID:            meterID,
			Namespace:     namespace,
			CreatedAt:     clock.Now(),
			UpdatedAt:     clock.Now(),
			Name:          "Flat per unit",
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))

	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 12, periodStart.Add(time.Minute))

	defer func() {
		err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
		s.Require().NoError(err, "meter adapter replace meters")
	}()

	featureFlatPerUnit := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterID:   lo.ToPtr(meterID),
	}))

	res, err := s.Charges.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.FiatCode(currency.USD),
			Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
				{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: namespace,
						Name:      "Test item1",
					}),
					ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},

					InvoiceAt: periodEnd,

					ManagedBy: billing.ManuallyManagedLine,

					Currency: currencyx.FiatCode(currency.USD),
					RateCardDiscounts: billing.Discounts{
						Percentage: &billing.PercentageDiscount{
							Percentage: models.NewPercentage(25),
						},
					},
					FeatureKey: featureFlatPerUnit.Key,
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.TieredPrice{
						Mode: productcatalog.GraduatedTieredPrice,
						Tiers: []productcatalog.PriceTier{
							{
								UnitPrice: &productcatalog.PriceTierUnitPrice{
									Amount: alpacadecimal.NewFromFloat(5),
								},
								UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
							},
							{
								UnitPrice: &productcatalog.PriceTierUnitPrice{
									Amount: alpacadecimal.NewFromFloat(4),
								},
							},
						},
					})),
				},
			}),
		})
	s.Require().NoError(err)
	s.Require().Len(res.Lines, 1)

	invoice1AsOf := lo.Must(time.Parse(time.RFC3339, "2026-01-15T00:00:00Z"))
	clock.FreezeTime(invoice1AsOf)
	defer clock.UnFreeze()

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
		AsOf:     lo.ToPtr(invoice1AsOf),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Empty(invoices[0].ValidationIssues)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

	invoiceLine := invoices[0].Lines.OrEmpty()[0]
	s.Require().Len(invoiceLine.DetailedLines, 2)
	s.Equal(float64(58), invoiceLine.Totals.Amount.InexactFloat64())
	s.Equal(float64(14.5), invoiceLine.Totals.DiscountsTotal.InexactFloat64())
	s.Equal(float64(43.5), invoiceLine.Totals.Total.InexactFloat64())

	clock.FreezeTime(invoices[0].DefaultCollectionAtForStandardInvoice())
	defer clock.UnFreeze()
	invoice1, err := s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
	s.Require().NoError(err)
	_, err = s.BillingService.ApproveInvoice(ctx, invoice1.GetInvoiceID())
	s.Require().NoError(err)

	s.Run("[invoice2] Retracted usage is rejected instead of reversing the booked discounts", func() {
		invoice2AsOf := periodEnd
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, -4, invoice2AsOf.Add(-30*time.Hour))
		clock.FreezeTime(invoice2AsOf)
		defer clock.UnFreeze()

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     lo.ToPtr(invoice2AsOf),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)

		invoice := invoices[0]
		s.Equal(billing.StandardInvoiceStatusDraftInvalidCreated, invoice.Status)

		s.Require().Len(invoice.ValidationIssues, 1)
		issue := invoice.ValidationIssues[0]
		s.Equal(string(usagebased.ErrCodeChargeTotalIsNegative), issue.Code)
		s.Equal(billing.ValidationIssueSeverityCritical, issue.Severity)

		s.Require().Len(invoice.Lines.OrEmpty(), 1)
		invoiceLine := invoice.Lines.OrEmpty()[0]
		s.Empty(invoiceLine.DetailedLines)
		s.True(invoiceLine.Totals.Total.IsZero())
		s.True(invoice.Totals.Total.IsZero())
	})
}

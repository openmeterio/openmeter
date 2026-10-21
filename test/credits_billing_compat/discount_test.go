package creditsbillingcompat

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type DiscountsTestSuite struct {
	BaseSuite
}

func TestDiscounts(t *testing.T) {
	suite.Run(t, new(DiscountsTestSuite))
}

func (s *DiscountsTestSuite) TestCorrelationIDHandling() {
	namespace := "ns-discounts-correlation-id"
	ctx := s.T().Context()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithProgressiveBilling())
	s.ProvisionDefaultTaxCodes(ctx, namespace)

	customerEntity := s.CreateTestCustomer(namespace, "test-customer")
	s.Require().NotNil(customerEntity)

	s.SetupAccounts(namespace, customerEntity.ID)

	meterSlug := "flat-per-unit"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ID:            meterID,
			Namespace:     namespace,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Name:          "Flat per unit",
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))

	periodStart := time.Now().Add(-time.Hour)
	periodEnd := time.Now().Add(time.Hour)

	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, periodStart.Add(-time.Minute))
	// Register some usage
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 10, periodStart.Add(time.Minute))

	defer func() {
		err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
		require.NoError(s.T(), err, "meter adapter replace meters")
	}()

	featureFlatPerUnit := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterID:   lo.ToPtr(meterID),
	}))

	// When we provision a new pending line with a 10% discount, without a correlation ID set

	var (
		discountCorrelationID string
		pendingLineChargeID   *string
	)

	s.Run("Creating new pending lines with discounts sets the correlationID", func() {
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
								Percentage: models.NewPercentage(10),
							},
						},
						FeatureKey: featureFlatPerUnit.Key,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						})),
					},
				}),
			})
		s.Require().NoError(err)
		s.Require().Len(res.Lines, 1)
		s.RequireChargeBasedGatheringLine(res.Lines[0], res.Lines[0].ChargeID)
		pendingLineChargeID = res.Lines[0].ChargeID

		// Then the freshly created line has a correlation ID set
		percentageDiscount := res.Lines[0].RateCardDiscounts.Percentage
		s.NotEmpty(percentageDiscount.CorrelationID)
		discountCorrelationID = percentageDiscount.CorrelationID

		_, err = ulid.Parse(percentageDiscount.CorrelationID)
		s.NoError(err, "the correlation ID is a valid ULID")
	})

	var draftInvoiceID billing.InvoiceID
	s.Run("Creating a draft invoice with progressive billing enabled retains the correlation ID", func() {
		// When the pending lines are invoiced in a progressive billing setup, the charge keeps the
		// correlation ID on the partially billed line.
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

		invoiceLine := invoices[0].Lines.OrEmpty()[0]
		s.Require().Nil(invoiceLine.SplitLineHierarchy)
		s.Require().Equal(pendingLineChargeID, invoiceLine.ChargeID)
		s.Equal(billing.LineEngineTypeChargeUsageBased, invoiceLine.Engine)

		// The partially billed line has the same correlation ID for the discount
		s.Equal(discountCorrelationID, invoiceLine.RateCardDiscounts.Percentage.CorrelationID)

		// An amount discount is also created, and it retains the same correlation ID
		s.Len(invoiceLine.DetailedLines, 1)
		detailedLine := invoiceLine.DetailedLines[0]

		require.Len(s.T(), detailedLine.AmountDiscounts, 1)

		amountDiscount := detailedLine.AmountDiscounts[0]

		s.Equal(billing.RatecardPercentageDiscountReason, amountDiscount.Reason.Type())
		pctDiscount, err := amountDiscount.Reason.AsRatecardPercentage()
		s.NoError(err)
		s.Equal(discountCorrelationID, pctDiscount.CorrelationID)

		// Output
		draftInvoiceID = invoices[0].GetInvoiceID()
	})

	s.Run("Editing a charge managed invoice line usage discount is rejected", func() {
		_, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      draftInvoiceID,
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.StandardInvoice) error {
				line := invoice.Lines.OrEmpty()[0]
				line.RateCardDiscounts.Usage = &billing.UsageDiscount{
					UsageDiscount: productcatalog.UsageDiscount{
						Quantity: alpacadecimal.NewFromFloat(10),
					},
				}

				return nil
			},
		})

		s.Require().Error(err)
		s.ErrorIs(err, billing.ErrCannotUpdateChargeManagedLine)
	})

	s.Run("Deleting the invoice works without errors", func() {
		invoice, err := s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
			Invoice:        draftInvoiceID,
			DeletionSource: billing.ChangeSourceAPIRequest,
		})
		s.NoError(err)
		s.Len(invoice.ValidationIssues, 0)
	})
}

func (s *DiscountsTestSuite) TestPercentageDiscountPersistenceAndCorrelationID() {
	for _, test := range []struct {
		name   string
		engine billing.LineEngineType
		price  *productcatalog.Price
	}{
		{
			name:   "flat fee",
			engine: billing.LineEngineTypeChargeFlatFee,
			price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
		},
		{
			name:   "usage based",
			engine: billing.LineEngineTypeChargeUsageBased,
			price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(10),
			}),
		},
	} {
		s.Run(test.name, func() {
			// given: equivalent gross charges with a percentage discount and a completed service period
			ctx := s.T().Context()
			namespace := s.GetUniqueNamespace("percentage-discount-persistence")
			period := timeutil.ClosedPeriod{
				From: time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 10, 12, 11, 0, 0, 0, time.UTC),
			}
			clock.FreezeTime(period.To.Add(time.Hour))
			defer clock.UnFreeze()

			sandboxApp := s.InstallSandboxApp(s.T(), namespace)
			s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithManualApproval())
			s.ProvisionDefaultTaxCodes(ctx, namespace)
			customerEntity := s.CreateTestCustomer(namespace, "test-customer")
			s.SetupAccounts(namespace, customerEntity.ID)

			var featureKey string
			if test.engine == billing.LineEngineTypeChargeUsageBased {
				usageFeature := s.SetupApiRequestsTotalFeature(ctx, namespace)
				defer usageFeature.Cleanup()
				featureKey = usageFeature.Feature.Key
				s.MockStreamingConnector.AddSimpleEvent(featureKey, 10, period.From.Add(time.Minute))
			}

			// when: the discount has no supplied correlation ID at charge creation
			pendingLines, err := s.Charges.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
				Customer: customerEntity.GetID(),
				Currency: currencyx.FiatCode(currency.USD),
				Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Discounted charge",
						}),
						ServicePeriod: period,
						InvoiceAt:     period.To,
						ManagedBy:     billing.ManuallyManagedLine,
						Price:         *test.price,
						FeatureKey:    featureKey,
						RateCardDiscounts: billing.Discounts{
							Percentage: &billing.PercentageDiscount{
								PercentageDiscount: productcatalog.PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
							},
						},
					},
				}}),
			})
			s.Require().NoError(err)
			s.Require().Len(pendingLines.Lines, 1)
			createdLine := pendingLines.Lines[0]
			s.RequireChargeBasedGatheringLine(createdLine, createdLine.ChargeID)
			s.Equal(test.engine, createdLine.Engine)
			s.Require().NotNil(createdLine.RateCardDiscounts.Percentage)
			discountCorrelationID := createdLine.RateCardDiscounts.Percentage.CorrelationID
			_, err = ulid.Parse(discountCorrelationID)
			s.Require().NoError(err, "the generated discount correlation ID must be a valid ULID")

			// then: gathering persistence retains the charge reference and generated discount lineage
			gatheringInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
				Invoice: pendingLines.Invoice.GetInvoiceID(),
				Expand:  billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
			})
			s.Require().NoError(err)
			s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)
			persistedLine := gatheringInvoice.Lines.OrEmpty()[0]
			s.RequireChargeBasedGatheringLine(persistedLine, createdLine.ChargeID)
			s.Equal(test.engine, persistedLine.Engine)
			s.Equal(createdLine.RateCardDiscounts, persistedLine.RateCardDiscounts)

			// when: the completed charge is invoiced and the standard invoice is reloaded
			invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: customerEntity.GetID(),
				AsOf:     lo.ToPtr(clock.Now()),
			})
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)
			invoice := invoices[0]
			reloadedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: invoice.GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)

			for _, result := range []struct {
				name    string
				invoice billing.StandardInvoice
			}{
				{name: "created invoice", invoice: invoice},
				{name: "reloaded invoice", invoice: reloadedInvoice},
			} {
				s.Run(result.name, func() {
					// then: both projections retain the original correlation ID and discounted amounts
					s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, result.invoice.Status)
					s.Require().Len(result.invoice.Lines.OrEmpty(), 1)
					invoiceLine := result.invoice.Lines.OrEmpty()[0]
					s.Equal(test.engine, invoiceLine.Engine)
					s.Equal(createdLine.ChargeID, invoiceLine.ChargeID)
					s.Nil(invoiceLine.SplitLineHierarchy)
					s.Require().NotNil(invoiceLine.RateCardDiscounts.Percentage)
					s.Equal(createdLine.RateCardDiscounts.Percentage, invoiceLine.RateCardDiscounts.Percentage)
					s.RequireTotals(billingtest.ExpectedTotals{
						Amount:         100,
						DiscountsTotal: 10,
						Total:          90,
					}, result.invoice.Totals)
					s.RequireTotals(billingtest.ExpectedTotals{
						Amount:         100,
						DiscountsTotal: 10,
						Total:          90,
					}, invoiceLine.Totals)
					s.Require().Len(invoiceLine.DetailedLines, 1)
					s.Require().Len(invoiceLine.DetailedLines[0].AmountDiscounts, 1)
					amountDiscount := invoiceLine.DetailedLines[0].AmountDiscounts[0]
					s.NotEmpty(amountDiscount.ID)
					s.Equal(billing.RatecardPercentageDiscountReason, amountDiscount.Reason.Type())
					reason, err := amountDiscount.Reason.AsRatecardPercentage()
					s.Require().NoError(err)
					s.Equal(discountCorrelationID, reason.CorrelationID)
					s.Equal(float64(10), reason.Percentage.InexactFloat64())
					s.Equal(float64(10), amountDiscount.Amount.InexactFloat64())
				})
			}
		})
	}
}

func (s *DiscountsTestSuite) TestFlatPriceUsageDiscountIsNotPersisted() {
	// given:
	// - a customer and a flat-priced pending line
	// - the line has a valid percentage discount and an invalid usage discount
	namespace := s.GetUniqueNamespace("ns-discounts-flat-price-usage")
	ctx := s.T().Context()
	customerEntity := s.CreateTestCustomer(namespace, "test-customer")
	now := time.Now()

	// when:
	// - the pending line is created
	_, err := s.Charges.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customerEntity.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: namespace,
						Name:      "invalid flat-price discount line",
					}),
					ServicePeriod: timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour)},
					InvoiceAt:     now.Add(time.Hour),
					ManagedBy:     billing.ManuallyManagedLine,
					RateCardDiscounts: billing.Discounts{
						Percentage: &billing.PercentageDiscount{
							PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
						},
						Usage: &billing.UsageDiscount{
							UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)},
						},
					},
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					})),
				},
			},
		}),
	})

	// then:
	// - validation rejects it before creating either an invoice or an invoice line
	s.ErrorIs(err, productcatalog.ErrUsageDiscountWithFlatPrice)

	lineCount, countErr := s.DBClient.BillingInvoiceLine.Query().
		Where(billinginvoiceline.Namespace(namespace)).
		Count(ctx)
	s.NoError(countErr)
	s.Zero(lineCount)

	invoiceCount, countErr := s.DBClient.BillingInvoice.Query().
		Where(billinginvoice.Namespace(namespace)).
		Count(ctx)
	s.NoError(countErr)
	s.Zero(invoiceCount)
}

func (s *DiscountsTestSuite) TestUnitDiscountProgressiveBilling() {
	// Given:
	// - we have a progressive billing enabled billing setup
	// - a unit priced pending line against a meter
	// - 110 unit discount applied to the line

	// When [invoice1]
	// - We do a partial invoicing with usage of 50 units
	// Then:
	// - The line in the draft invoice will have 0 quantity, 50 metered quantity
	// - The line will have an unit line discount with quantity 50, and preLinePeriodQuantity 0

	// When [invoice2]
	// - we do a second partial invoicing with the usage of 75 units
	// Then:
	// - The line in the draft invoice will have 15 quantity, 75 metered quantity
	// - The line will have an unit line discount with quantity 60, and preLinePeriodQuantity 50

	// When [invoice3]
	// - we do a final invoice with the usage of 30 units
	// Then:
	// - The line in the draft invoice will have 30 quantity, 30 metered quantity
	// - The line will not have an unit line discount
	namespace := "ns-discounts-usage-progressive"
	ctx := s.T().Context()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
		profile.WorkflowConfig.Invoicing.ProgressiveBilling = true
	}))
	s.ProvisionDefaultTaxCodes(ctx, namespace)

	customerEntity := s.CreateTestCustomer(namespace, "test-customer")
	s.Require().NotNil(customerEntity)

	s.SetupAccounts(namespace, customerEntity.ID)

	meterSlug := "flat-per-unit"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: meterID,
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Flat per unit",
			},
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))

	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now().Add(-time.Hour)

	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, periodStart.Add(-time.Minute))

	defer func() {
		err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
		require.NoError(s.T(), err, "meter adapter replace meters")
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
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Test item1",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},

						InvoiceAt: periodEnd,

						ManagedBy: billing.ManuallyManagedLine,

						Currency: currencyx.FiatCode(currency.USD),
						RateCardDiscounts: billing.Discounts{
							Usage: &billing.UsageDiscount{
								UsageDiscount: productcatalog.UsageDiscount{
									Quantity: alpacadecimal.NewFromInt(110),
								},
							},
						},
						FeatureKey: featureFlatPerUnit.Key,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						})),
					},
				},
			}),
		})
	s.Require().NoError(err)
	s.Require().Len(res.Lines, 1)
	s.RequireChargeBasedGatheringLine(res.Lines[0], res.Lines[0].ChargeID)
	chargeID := *res.Lines[0].ChargeID

	require.NotNil(s.T(), res.Lines[0].RateCardDiscounts.Usage)
	require.NotEmpty(s.T(), res.Lines[0].RateCardDiscounts.Usage.CorrelationID)
	discountCorrelationID := res.Lines[0].RateCardDiscounts.Usage.CorrelationID

	var invoice1ID billing.InvoiceID
	s.Run("[invoice1] Creating a draft invoice with 50 usage", func() {
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, 50, periodStart.Add(time.Minute))
		invoice1AsOf := periodStart.Add(time.Hour)

		// The discount correlation ID is retained across realizations of the same charge.
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     lo.ToPtr(invoice1AsOf),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

		invoice1ID = invoices[0].GetInvoiceID()
		invoiceLine := invoices[0].Lines.OrEmpty()[0]
		s.Require().Nil(invoiceLine.SplitLineHierarchy)
		s.Require().NotNil(invoiceLine.ChargeID)
		s.Require().Equal(chargeID, *invoiceLine.ChargeID)

		// The line has 0 quantity, 50 metered quantity
		require.Equal(s.T(), float64(0), invoiceLine.UsageBased.Quantity.InexactFloat64())
		require.Equal(s.T(), float64(50), invoiceLine.UsageBased.MeteredQuantity.InexactFloat64())

		require.Equal(s.T(), float64(0), invoiceLine.UsageBased.PreLinePeriodQuantity.InexactFloat64())
		require.Equal(s.T(), float64(0), invoiceLine.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())

		s.Require().Len(invoiceLine.Discounts.Usage, 1)
		usageDiscount := invoiceLine.Discounts.Usage[0]

		// The usage discount has quantity 50, and preLinePeriodQuantity 0
		require.Equal(s.T(), float64(50), usageDiscount.Quantity.InexactFloat64())
		require.Nil(s.T(), usageDiscount.PreLinePeriodQuantity)
		// Sanity check discount data
		require.Equal(s.T(), billing.RatecardUsageDiscountReason, usageDiscount.Reason.Type())
		reason, err := usageDiscount.Reason.AsRatecardUsage()
		require.NoError(s.T(), err)

		require.Equal(s.T(), discountCorrelationID, reason.CorrelationID)

		// The detailed line does not exists, as we had no usage
		require.Len(s.T(), invoiceLine.DetailedLines, 0)
	})

	// Complete the first realization before collecting the next part of the charge.
	_, err = s.BillingService.ApproveInvoice(ctx, invoice1ID)
	s.Require().NoError(err)

	var invoice2ID billing.InvoiceID
	s.Run("[invoice2] Creating a draft invoice with 75 usage", func() {
		invoice2AsOf := periodStart.Add(time.Hour * 3)
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, 75, invoice2AsOf.Add(-time.Minute))

		// The discount correlation ID is retained across realizations of the same charge.
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     lo.ToPtr(invoice2AsOf),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

		invoice2ID = invoices[0].GetInvoiceID()
		invoiceLine := invoices[0].Lines.OrEmpty()[0]
		s.Require().Nil(invoiceLine.SplitLineHierarchy)
		s.Require().NotNil(invoiceLine.ChargeID)
		s.Require().Equal(chargeID, *invoiceLine.ChargeID)

		// The line has 0 quantity, 50 metered quantity
		require.Equal(s.T(), float64(15), invoiceLine.UsageBased.Quantity.InexactFloat64())
		require.Equal(s.T(), float64(75), invoiceLine.UsageBased.MeteredQuantity.InexactFloat64())

		require.Equal(s.T(), float64(0), invoiceLine.UsageBased.PreLinePeriodQuantity.InexactFloat64())
		require.Equal(s.T(), float64(50), invoiceLine.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())

		s.Require().Len(invoiceLine.Discounts.Usage, 1)
		usageDiscount := invoiceLine.Discounts.Usage[0]

		// The usage discount has quantity 50, and preLinePeriodQuantity 0
		require.Equal(s.T(), float64(60), usageDiscount.Quantity.InexactFloat64())
		require.Equal(s.T(), float64(50), usageDiscount.PreLinePeriodQuantity.InexactFloat64())
		// Sanity check discount data
		require.Equal(s.T(), billing.RatecardUsageDiscountReason, usageDiscount.Reason.Type())
		reason, err := usageDiscount.Reason.AsRatecardUsage()
		require.NoError(s.T(), err)

		require.Equal(s.T(), discountCorrelationID, reason.CorrelationID)
		require.Equal(s.T(), float64(110), reason.Quantity.InexactFloat64())

		// The detailed line exists and has a usage of 15
		require.Len(s.T(), invoiceLine.DetailedLines, 1)
		detailedLine := invoiceLine.DetailedLines[0]
		require.Equal(s.T(), float64(15), detailedLine.Quantity.InexactFloat64())
	})

	_, err = s.BillingService.ApproveInvoice(ctx, invoice2ID)
	s.Require().NoError(err)

	s.Run("[invoice3] Creating a draft invoice with 30 usage", func() {
		invoice3AsOf := periodEnd
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, 30, invoice3AsOf.Add(-time.Minute))

		// The discount correlation ID is retained across realizations of the same charge.
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     lo.ToPtr(invoice3AsOf),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

		invoiceLine := invoices[0].Lines.OrEmpty()[0]
		s.Require().Nil(invoiceLine.SplitLineHierarchy)
		s.Require().NotNil(invoiceLine.ChargeID)
		s.Require().Equal(chargeID, *invoiceLine.ChargeID)

		// The line has 0 quantity, 50 metered quantity
		require.Equal(s.T(), float64(30), invoiceLine.UsageBased.Quantity.InexactFloat64())
		require.Equal(s.T(), float64(30), invoiceLine.UsageBased.MeteredQuantity.InexactFloat64())

		require.Equal(s.T(), float64(15), invoiceLine.UsageBased.PreLinePeriodQuantity.InexactFloat64())
		require.Equal(s.T(), float64(125), invoiceLine.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())

		s.Len(invoiceLine.Discounts.Usage, 0)

		// The detailed line exists and has a usage of 30
		require.Len(s.T(), invoiceLine.DetailedLines, 1)
		detailedLine := invoiceLine.DetailedLines[0]
		require.Equal(s.T(), float64(30), detailedLine.Quantity.InexactFloat64())
	})
}

package billing

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type DiscountsTestSuite struct {
	BaseSuite
}

func TestDiscounts(t *testing.T) {
	suite.Run(t, new(DiscountsTestSuite))
}

func (s *DiscountsTestSuite) TestCorrelationIDHandling() {
	namespace := "ns-discounts-correlation-id"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithProgressiveBilling())

	customerEntity := s.CreateTestCustomer(namespace, "test-customer")
	s.NotNil(customerEntity)

	meterSlug := "flat-per-unit"
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
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
		MeterSlug: lo.ToPtr(meterSlug),
	}))

	// When we provision a new pending line with a 10% discount, without a correlation ID set

	var discountCorrelationID string
	s.Run("Creating new pending lines with discounts sets the correlationID", func() {
		res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
			billing.CreatePendingInvoiceLinesInput{
				Customer: customerEntity.GetID(),
				Currency: currencyx.Code(currency.USD),
				Lines: []*billing.StandardLine{
					{
						StandardLineBase: billing.StandardLineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Namespace: namespace,
								Name:      "Test item1",
							}),
							Period: billing.Period{Start: periodStart, End: periodEnd},

							InvoiceAt: periodEnd,

							ManagedBy: billing.ManuallyManagedLine,

							Currency: currencyx.Code(currency.USD),
							RateCardDiscounts: billing.Discounts{
								Percentage: &billing.PercentageDiscount{
									PercentageDiscount: productcatalog.PercentageDiscount{
										Percentage: models.NewPercentage(10),
									},
								},
							},
						},
						UsageBased: &billing.UsageBasedLine{
							FeatureKey: featureFlatPerUnit.Key,
							Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
								Amount: alpacadecimal.NewFromFloat(100),
							}),
						},
					},
				},
			})
		s.NoError(err)
		s.Len(res.Lines, 1)

		// Then the freshly created line has a correlation ID set
		percentageDiscount := res.Lines[0].RateCardDiscounts.Percentage
		s.NotEmpty(percentageDiscount.CorrelationID)
		discountCorrelationID = percentageDiscount.CorrelationID

		_, err = ulid.Parse(percentageDiscount.CorrelationID)
		s.NoError(err, "the correlation ID is a valid ULID")
	})

	var draftInvoiceID billing.InvoiceID
	s.Run("Creating a draft invoice with progressive billing enabled retains the correlation ID", func() {
		// When the pending lines are invoiced in a progressive billing setup, the correlation ID is retained
		// between the split lines.
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		s.Len(invoices[0].Lines.OrEmpty(), 1)

		invoiceLine := invoices[0].Lines.OrEmpty()[0]
		s.NotNil(invoiceLine.SplitLineHierarchy)

		// Root line has the same correlation ID for the discount
		s.Equal(discountCorrelationID, invoiceLine.RateCardDiscounts.Percentage.CorrelationID)

		// Split lines have the same correlation ID for the discount
		s.Len(invoiceLine.SplitLineHierarchy.Lines, 2)
		for _, lineWithInvoice := range invoiceLine.SplitLineHierarchy.Lines {
			s.Equal(discountCorrelationID, lineWithInvoice.Line.RateCardDiscounts.Percentage.CorrelationID)
		}

		// The split line group has the same correlation ID for the discount
		s.Equal(discountCorrelationID, invoiceLine.SplitLineHierarchy.Group.RatecardDiscounts.Percentage.CorrelationID)

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
		draftInvoiceID = invoices[0].InvoiceID()
	})

	s.Run("Editing an invoice and adding a new discount generates a new correlation ID", func() {
		editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: draftInvoiceID,
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
		s.NoError(err)
		s.NotNil(editedInvoice)

		s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, editedInvoice.Status)

		rcDiscounts := editedInvoice.Lines.OrEmpty()[0].RateCardDiscounts
		s.NotNil(rcDiscounts)

		s.Equal(discountCorrelationID, rcDiscounts.Percentage.CorrelationID)
		s.NotEqual(discountCorrelationID, rcDiscounts.Usage.CorrelationID)
		s.NotEmpty(rcDiscounts.Usage.CorrelationID)
	})

	s.Run("Deleting the invoice works without errors", func() {
		invoice, err := s.BillingService.DeleteInvoice(ctx, draftInvoiceID)
		s.NoError(err)
		s.Len(invoice.ValidationIssues, 0)
	})
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
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
		profile.WorkflowConfig.Invoicing.ProgressiveBilling = true
	}))

	customerEntity := s.CreateTestCustomer(namespace, "test-customer")
	s.NotNil(customerEntity)

	meterSlug := "flat-per-unit"
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
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
		MeterSlug: lo.ToPtr(meterSlug),
	}))

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []*billing.StandardLine{
				{
					StandardLineBase: billing.StandardLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Test item1",
						}),
						Period: billing.Period{Start: periodStart, End: periodEnd},

						InvoiceAt: periodEnd,

						ManagedBy: billing.ManuallyManagedLine,

						Currency: currencyx.Code(currency.USD),
						RateCardDiscounts: billing.Discounts{
							Usage: &billing.UsageDiscount{
								UsageDiscount: productcatalog.UsageDiscount{
									Quantity: alpacadecimal.NewFromInt(110),
								},
							},
						},
					},
					UsageBased: &billing.UsageBasedLine{
						FeatureKey: featureFlatPerUnit.Key,
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						}),
					},
				},
			},
		})
	s.NoError(err)
	s.Len(res.Lines, 1)

	require.NotNil(s.T(), res.Lines[0].RateCardDiscounts.Usage)
	require.NotEmpty(s.T(), res.Lines[0].RateCardDiscounts.Usage.CorrelationID)
	discountCorrelationID := res.Lines[0].RateCardDiscounts.Usage.CorrelationID

	s.Run("[invoice1] Creating a draft invoice with 50 usage", func() {
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, 50, periodStart.Add(time.Minute))
		invoice1AsOf := periodStart.Add(time.Hour)

		// When the pending lines are invoiced in a progressive billing setup, the correlation ID is retained
		// between the split lines.
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     lo.ToPtr(invoice1AsOf),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		s.Len(invoices[0].Lines.OrEmpty(), 1)

		invoiceLine := invoices[0].Lines.OrEmpty()[0]
		s.NotNil(invoiceLine.SplitLineHierarchy)

		// The line has 0 quantity, 50 metered quantity
		require.Equal(s.T(), float64(0), invoiceLine.UsageBased.Quantity.InexactFloat64())
		require.Equal(s.T(), float64(50), invoiceLine.UsageBased.MeteredQuantity.InexactFloat64())

		require.Equal(s.T(), float64(0), invoiceLine.UsageBased.PreLinePeriodQuantity.InexactFloat64())
		require.Equal(s.T(), float64(0), invoiceLine.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())

		s.Len(invoiceLine.Discounts.Usage, 1)
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

	s.Run("[invoice2] Creating a draft invoice with 75 usage", func() {
		invoice2AsOf := periodStart.Add(time.Hour * 3)
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, 75, invoice2AsOf.Add(-time.Minute))

		// When the pending lines are invoiced in a progressive billing setup, the correlation ID is retained
		// between the split lines.
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     lo.ToPtr(invoice2AsOf),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		s.Len(invoices[0].Lines.OrEmpty(), 1)

		invoiceLine := invoices[0].Lines.OrEmpty()[0]
		s.NotNil(invoiceLine.SplitLineHierarchy)

		// The line has 0 quantity, 50 metered quantity
		require.Equal(s.T(), float64(15), invoiceLine.UsageBased.Quantity.InexactFloat64())
		require.Equal(s.T(), float64(75), invoiceLine.UsageBased.MeteredQuantity.InexactFloat64())

		require.Equal(s.T(), float64(0), invoiceLine.UsageBased.PreLinePeriodQuantity.InexactFloat64())
		require.Equal(s.T(), float64(50), invoiceLine.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())

		s.Len(invoiceLine.Discounts.Usage, 1)
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

	s.Run("[invoice3] Creating a draft invoice with 30 usage", func() {
		invoice3AsOf := periodEnd
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, 30, invoice3AsOf.Add(-time.Minute))

		// When the pending lines are invoiced in a progressive billing setup, the correlation ID is retained
		// between the split lines.
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     lo.ToPtr(invoice3AsOf),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		s.Len(invoices[0].Lines.OrEmpty(), 1)

		invoiceLine := invoices[0].Lines.OrEmpty()[0]
		s.NotNil(invoiceLine.SplitLineHierarchy)

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

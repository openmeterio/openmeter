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

	defaultProfileSettings := MinimalCreateProfileInputTemplate
	defaultProfileSettings.Default = true
	defaultProfileSettings.Namespace = namespace
	defaultProfileSettings.WorkflowConfig.Invoicing.ProgressiveBilling = true

	s.InstallSandboxApp(s.T(), namespace)

	defaultProfile, err := s.BillingService.CreateProfile(ctx, defaultProfileSettings)
	s.NoError(err)
	s.NotNil(defaultProfile)

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
		err = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
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
			billing.CreateInvoiceLinesInput{
				Namespace: namespace,
				Lines: []billing.LineWithCustomer{
					{
						Line: billing.Line{
							LineBase: billing.LineBase{
								Namespace: namespace,
								Period:    billing.Period{Start: periodStart, End: periodEnd},

								InvoiceAt: periodEnd,

								Type:      billing.InvoiceLineTypeUsageBased,
								ManagedBy: billing.ManuallyManagedLine,

								Name:     "Test item1",
								Currency: currencyx.Code(currency.USD),
								RateCardDiscounts: billing.Discounts{
									billing.NewDiscountFrom(billing.PercentageDiscount{
										PercentageDiscount: productcatalog.PercentageDiscount{
											Percentage: models.NewPercentage(10),
										},
									}),
								},
							},
							UsageBased: &billing.UsageBasedLine{
								FeatureKey: featureFlatPerUnit.Key,
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(100),
								}),
							},
						},
						CustomerID: customerEntity.ID,
					},
				},
			})
		s.NoError(err)
		s.Len(res, 1)

		// Then the freshly created line has a correlation ID set
		percentageDiscount := lo.Must(res[0].RateCardDiscounts[0].AsPercentage())
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
		s.NotNil(invoiceLine.ProgressiveLineHierarchy)

		// Root line has the same correlation ID for the discount
		s.Equal(discountCorrelationID, getDiscountCorrelationID(invoiceLine.RateCardDiscounts[0]))

		// Split lines have the same correlation ID for the discount
		s.Len(invoiceLine.ProgressiveLineHierarchy.Children, 2)
		for _, child := range invoiceLine.ProgressiveLineHierarchy.Children {
			s.Equal(discountCorrelationID, getDiscountCorrelationID(child.Line.RateCardDiscounts[0]))
		}

		// An amount discount is also created, and it retains the same correlation ID
		s.Len(invoiceLine.Children.OrEmpty(), 1)
		detailedLine := invoiceLine.Children.OrEmpty()[0]

		amountDiscount := lo.Must(detailedLine.Discounts[0].AsAmount())
		s.Equal(billing.LineDiscountReasonRatecardDiscount, amountDiscount.Reason)
		s.NotNil(amountDiscount.SourceDiscount)
		s.Equal(discountCorrelationID, getDiscountCorrelationID(*amountDiscount.SourceDiscount))

		// Output
		draftInvoiceID = invoices[0].InvoiceID()
	})

	s.Run("Editing an invoice and adding a new discount generates a new correlation ID", func() {
		editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: draftInvoiceID,
			EditFn: func(invoice *billing.Invoice) error {
				line := invoice.Lines.OrEmpty()[0]
				line.RateCardDiscounts = append(line.RateCardDiscounts, billing.NewDiscountFrom(billing.PercentageDiscount{
					PercentageDiscount: productcatalog.PercentageDiscount{
						Percentage: models.NewPercentage(20),
					},
				}))

				return nil
			},
		})
		s.NoError(err)
		s.NotNil(editedInvoice)

		rcDiscounts := editedInvoice.Lines.OrEmpty()[0].RateCardDiscounts
		s.Len(rcDiscounts, 2)

		s.Equal(discountCorrelationID, getDiscountCorrelationID(rcDiscounts[0]))
		s.NotEqual(discountCorrelationID, getDiscountCorrelationID(rcDiscounts[1]))
	})
}

func getDiscountCorrelationID(discount billing.Discount) string {
	switch discount.Type() {
	case productcatalog.PercentageDiscountType:
		return lo.Must(discount.AsPercentage()).CorrelationID
	case productcatalog.UsageDiscountType:
		return lo.Must(discount.AsUsage()).CorrelationID
	default:
		return ""
	}
}

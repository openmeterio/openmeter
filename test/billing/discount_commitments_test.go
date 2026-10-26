package billing

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *DiscountsTestSuite) TestPercentageDiscountProgressiveCommitments() {
	type invoiceSlice struct {
		endAfter  time.Duration
		quantity  float64
		amount    float64
		discounts float64
		charges   float64
		total     float64
	}

	for _, tc := range []struct {
		name        string
		commitments productcatalog.Commitments
		slices      []invoiceSlice
	}{
		{
			name: "minimum spend tops up discounted prior usage",
			commitments: productcatalog.Commitments{
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
			},
			slices: []invoiceSlice{
				{endAfter: 8 * time.Hour, quantity: 10, amount: 100, discounts: 40, total: 60},
				{endAfter: 24 * time.Hour, charges: 40, total: 40},
			},
		},
		{
			name: "maximum spend caps discounted usage across repeated invoices",
			commitments: productcatalog.Commitments{
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
			},
			slices: []invoiceSlice{
				{endAfter: 8 * time.Hour, quantity: 10, amount: 100, discounts: 40, total: 60},
				{endAfter: 16 * time.Hour, quantity: 10, amount: 100, discounts: 60, total: 40},
				{endAfter: 24 * time.Hour, quantity: 10, amount: 100, discounts: 100, total: 0},
			},
		},
	} {
		s.Run(tc.name, func() {
			// Given a progressive unit price of 10, a 40% discount, and a spend commitment of 100.
			ctx := s.T().Context()
			namespace := s.GetUniqueNamespace("discounted-progressive-commitment")
			period := timeutil.ClosedPeriod{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			}
			clock.FreezeTime(period.To.Add(time.Hour))
			defer clock.UnFreeze()

			sandboxApp := s.InstallSandboxApp(s.T(), namespace)
			customerEntity := s.CreateTestCustomer(namespace, "test-customer")
			s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithProgressiveBilling())
			s.TaxCodeEnv.ProvisionProviderDefaultTaxCode(s.T(), namespace)
			testFeature := s.SetupApiRequestsTotalFeature(ctx, namespace)
			defer testFeature.Cleanup()

			_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
				Customer: customerEntity.GetID(),
				Currency: currencyx.FiatCode(currency.USD),
				Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
					{
						GatheringLineBase: billing.GatheringLineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Name: "discounted usage",
							}),
							ServicePeriod: period,
							InvoiceAt:     period.To,
							ManagedBy:     billing.ManuallyManagedLine,
							FeatureKey:    testFeature.Feature.Key,
							Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
								Amount:      alpacadecimal.NewFromInt(10),
								Commitments: tc.commitments,
							})),
							RateCardDiscounts: billing.Discounts{
								Percentage: &billing.PercentageDiscount{
									PercentageDiscount: productcatalog.PercentageDiscount{
										Percentage: models.NewPercentage(40),
									},
								},
							},
						},
					},
				}),
			})
			s.Require().NoError(err)

			var totalBilled alpacadecimal.Decimal
			var groupID string
			for idx, expected := range tc.slices {
				// When each slice is collected, the engine reloads the earlier invoices from storage.
				asOf := period.From.Add(expected.endAfter)
				s.MockStreamingConnector.AddSimpleEvent(testFeature.Feature.Key, expected.quantity, asOf.Add(-time.Hour))
				invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
					Customer: customerEntity.GetID(),
					AsOf:     &asOf,
				})
				s.Require().NoError(err)
				s.Require().Len(invoices, 1)
				s.Require().Len(invoices[0].Lines.OrEmpty(), 1)
				stdLine := invoices[0].Lines.OrEmpty()[0]
				s.Require().NotNil(stdLine.SplitLineGroupID)
				if idx == 0 {
					groupID = *stdLine.SplitLineGroupID
				}

				// Then each invoice uses net prior spend, including when the maximum was already reached.
				s.Equal(groupID, *stdLine.SplitLineGroupID)
				s.Equal(expected.amount, stdLine.Totals.Amount.InexactFloat64(), "slice %d gross amount", idx+1)
				s.Equal(expected.discounts, stdLine.Totals.DiscountsTotal.InexactFloat64(), "slice %d discounts", idx+1)
				s.Equal(expected.charges, stdLine.Totals.ChargesTotal.InexactFloat64(), "slice %d minimum top-up", idx+1)
				s.Equal(expected.total, stdLine.Totals.Total.InexactFloat64(), "slice %d net total", idx+1)
				totalBilled = totalBilled.Add(stdLine.Totals.Total)
			}

			s.Equal(float64(100), totalBilled.InexactFloat64())
		})
	}
}

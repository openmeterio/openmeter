package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargedetailedline "github.com/openmeterio/openmeter/openmeter/billing/charges/models/detailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *CreditThenInvoiceTestSuite) TestUsageBasedAmountDiscountsDeferProgressiveBilling() {
	for _, test := range []struct {
		name                    string
		price                   *productcatalog.Price
		discounts               billing.Discounts
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
				Amount: alpacadecimal.NewFromInt(2),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(60)),
				},
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
	} {
		s.Run(test.name, func() {
			// given: a discounted charge has usage and progressive billing enabled
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
			s.MockStreamingConnector.AddSimpleEvent(feature.Feature.Key, test.usageSent, period.From.Add(time.Hour))
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

			// then: the charge remains uninvoiced without creating a partial run
			s.Require().ErrorIs(err, billing.ErrInvoiceCreateNoLines)
			s.Empty(invoices)
			waitingCharge := s.mustGetUsageBasedChargeByIDWithExpands(charge.GetChargeID(), meta.Expands{
				meta.ExpandRealizations,
			})
			s.Empty(waitingCharge.Realizations)
			s.Nil(waitingCharge.State.CurrentRealizationRunID)

			// when: billing snapshots the complete service period at its end
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
			s.Equal(period, line.Period)
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

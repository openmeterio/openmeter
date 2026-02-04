package billing

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type UBPFlatFeeLineTestSuite struct {
	BaseSuite
}

func TestUBPFlatFeeLine(t *testing.T) {
	suite.Run(t, new(UBPFlatFeeLineTestSuite))
}

func (s *UBPFlatFeeLineTestSuite) TestPendingLineCreation() {
	namespace := "ns-ubpff-line-create"
	ctx := context.Background()

	clockBase := lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	cust := s.CreateTestCustomer(namespace, "test")
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	// Given we have a customer
	// When we create a pending fee line using the usage based flat fee line
	// Then the gathering invoice should be created

	period := billing.Period{
		Start: clockBase,
		End:   clockBase.Add(time.Hour * 24),
	}

	s.Run("should create a pending line", func() {
		lineIn := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
			Period:    period,
			InvoiceAt: period.End,

			Currency:      "USD",
			Name:          "test in arrears",
			PerUnitAmount: alpacadecimal.NewFromInt(100),
			PaymentTerm:   productcatalog.InArrearsPaymentTerm,
		})

		res, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: cust.GetID(),
			Currency: "USD",
			Lines: []billing.GatheringLine{
				lineIn,
			},
		})

		s.NoError(err)
		s.NotNil(res)

		line := res.Lines[0]
		expected, err := lineIn.Clone()
		s.NoError(err)

		// Let's add fields coming from the line creation
		expected.Namespace = cust.Namespace
		expected.InvoiceID = res.Invoice.ID
		expected.ID = line.ID
		expected.CreatedAt = line.CreatedAt
		expected.UpdatedAt = line.UpdatedAt
		expected.UBPConfigID = line.UBPConfigID

		ExpectJSONEqual(s.T(),
			lo.Must(expected.WithoutDBState()),
			lo.Must(line.WithoutDBState()))
	})

	// Given the line on gathering invoice is created
	// When we try to create a progressively billed line
	// Then the line should not be created
	s.Run("should not create a progressively billed line", func() {
		_, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer:                   cust.GetID(),
			ProgressiveBillingOverride: lo.ToPtr(true),
		})
		s.Error(err)
		s.ErrorIs(err, billing.ErrInvoiceCreateNoLines)
	})

	// Given the line on gathering invoice is created
	// When we create a draft invoice
	// Then the invoice should be created and should contain the line with it's detailed lines
	s.Run("should create a draft invoice", func() {
		clock.SetTime(period.End)

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
		})
		s.NoError(err)

		s.Len(invoices, 1)
		invoice := invoices[0]
		lines := invoice.Lines.MustGet()
		s.Len(lines, 1)
		line := lines[0]

		s.Len(line.DetailedLines, 1)
		detailedLine := line.DetailedLines[0]

		// Let's validate the detailed line
		s.Equal(float64(1), detailedLine.Quantity.InexactFloat64())
		s.Equal(float64(100), detailedLine.PerUnitAmount.InexactFloat64())
		s.Equal(productcatalog.InArrearsPaymentTerm, detailedLine.PaymentTerm)
		s.Equal("test in arrears", detailedLine.Name)

		// Let's validate the totals
		requireTotals(s.T(), expectedTotals{
			Amount: 100,
			Total:  100,
		}, line.Totals)

		requireTotals(s.T(), expectedTotals{
			Amount: 100,
			Total:  100,
		}, detailedLine.Totals)
	})
}

func (s *UBPFlatFeeLineTestSuite) TestPercentageDiscount() {
	namespace := "ns-ubpff-percentage-discount"
	ctx := context.Background()

	clock.SetTime(lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")))
	defer clock.ResetTime()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	cust := s.CreateTestCustomer(namespace, "test")
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	// Given we have a customer
	// When we create a pending fee line using the usage based flat fee line with a percentage discount
	// Then the final invoice should contain the amount details

	period := billing.Period{
		Start: clock.Now(),
		End:   clock.Now().Add(time.Hour * 24),
	}

	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: "USD",
		Lines: []billing.GatheringLine{
			billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Period:    period,
				InvoiceAt: period.End,

				Name:          "test in arrears",
				PerUnitAmount: alpacadecimal.NewFromInt(200),
				PaymentTerm:   productcatalog.InArrearsPaymentTerm,

				RateCardDiscounts: billing.Discounts{
					Percentage: &billing.PercentageDiscount{
						PercentageDiscount: productcatalog.PercentageDiscount{
							Percentage: models.NewPercentage(50),
						},
					},
				},
			}),
		},
	})
	s.NoError(err)

	clock.SetTime(period.End)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)

	s.Len(invoices, 1)
	invoice := invoices[0]
	lines := invoice.Lines.MustGet()
	s.Len(lines, 1)
	line := lines[0]

	s.Len(line.DetailedLines, 1)
	detailedLine := line.DetailedLines[0]

	// Let's validate the lines

	amountDiscounts := detailedLine.AmountDiscounts
	s.Len(amountDiscounts, 1)
	amountDiscount := amountDiscounts[0]
	s.Equal(float64(100), amountDiscount.Amount.InexactFloat64())
	s.Equal(float64(0), amountDiscount.RoundingAmount.InexactFloat64())

	requireTotals(s.T(), expectedTotals{
		Amount:         200,
		DiscountsTotal: 100,
		Total:          100,
	}, line.Totals)

	requireTotals(s.T(), expectedTotals{
		Amount:         200,
		DiscountsTotal: 100,
		Total:          100,
	}, detailedLine.Totals)
}

func (s *UBPFlatFeeLineTestSuite) TestValidations() {
	namespace := "ns-ubpff-validations"
	ctx := context.Background()

	clock.SetTime(lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")))
	defer clock.ResetTime()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	cust := s.CreateTestCustomer(namespace, "test")
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	period := billing.Period{
		Start: clock.Now(),
		End:   clock.Now().Add(time.Hour * 24),
	}

	s.Run("should not create line with usage discount", func() {
		_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: cust.GetID(),
			Currency: "USD",
			Lines: []billing.GatheringLine{
				billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
					Period:    period,
					InvoiceAt: period.End,

					Name:          "test in arrears",
					PerUnitAmount: alpacadecimal.NewFromInt(100),
					PaymentTerm:   productcatalog.InArrearsPaymentTerm,

					RateCardDiscounts: billing.Discounts{
						Usage: &billing.UsageDiscount{
							UsageDiscount: productcatalog.UsageDiscount{
								Quantity: alpacadecimal.NewFromInt(1),
							},
						},
					},
				}),
			},
		})
		s.Error(err)
	})

	s.Run("empty period is allowed", func() {
		_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: cust.GetID(),
			Currency: "USD",
			Lines: []billing.GatheringLine{
				billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
					Period: billing.Period{
						Start: period.Start,
						End:   period.Start,
					},
					InvoiceAt: period.Start,

					Name:          "test in arrears",
					PerUnitAmount: alpacadecimal.NewFromInt(100),
					PaymentTerm:   productcatalog.InArrearsPaymentTerm,
				}),
			},
		})
		s.NoError(err)
	})
}

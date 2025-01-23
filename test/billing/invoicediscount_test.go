package billing

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type InvoiceDiscountTestSuite struct {
	BaseSuite
}

func TestInvoiceDiscounts(t *testing.T) {
	suite.Run(t, new(InvoiceDiscountTestSuite))
}

type invoiceDiscountAdapter interface {
	GetInvoiceDiscount(ctx context.Context, id billing.InvoiceDiscountID) (billing.InvoiceDiscount, error)
}

func (s *InvoiceDiscountTestSuite) TestInvoiceDiscountSync() {
	namespace := "ns-invoice-discount-test"
	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)
	periodEnd := now.Add(-time.Hour)
	periodStart := periodEnd.Add(-time.Hour * 24 * 30)
	issueAt := now.Add(-time.Minute)

	_ = s.InstallSandboxApp(s.T(), namespace)

	customerEntity := s.CreateTestCustomer(namespace, "test")

	minimalCreateProfileInput := MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = namespace

	_, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)
	s.NoError(err)

	discountGetter, ok := s.BillingAdapter.(invoiceDiscountAdapter)
	s.True(ok)

	lines, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreateInvoiceLinesInput{
		Namespace: namespace,
		Lines: []billing.LineWithCustomer{
			{
				Line: billing.Line{
					LineBase: billing.LineBase{
						Namespace: namespace,
						Period:    billing.Period{Start: periodStart, End: periodEnd},
						InvoiceAt: issueAt,
						ManagedBy: billing.ManuallyManagedLine,

						Type: billing.InvoiceLineTypeFee,

						Name:     "Test item - USD",
						Currency: currencyx.Code(currency.USD),
					},
					FlatFee: &billing.FlatFeeLine{
						PerUnitAmount: alpacadecimal.NewFromFloat(100),
						Quantity:      alpacadecimal.NewFromFloat(1),
						Category:      billing.FlatFeeCategoryRegular,
						PaymentTerm:   productcatalog.InAdvancePaymentTerm,
					},
				},
				CustomerID: customerEntity.ID,
			},
		},
	})
	s.NoError(err)
	s.Len(lines, 1)

	s.Run("Add discount to invoice", func() {
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: billing.InvoiceID{
				Namespace: namespace,
				ID:        lines[0].InvoiceID,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)

		updatedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: gatheringInvoice.InvoiceID(),
			EditFn: func(invoice *billing.Invoice) error {
				invoice.Discounts.Append(
					billing.NewInvoiceDiscountFrom(
						billing.InvoiceDiscountPercentage{
							InvoiceDiscountBase: billing.InvoiceDiscountBase{
								ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
									Namespace: namespace,
									Name:      "Test discount",
								}),
								InvoiceID: gatheringInvoice.ID,
								Type:      billing.PercentageDiscountType,
								LineIDs:   []string{lines[0].ID},
							},
							Percentage: alpacadecimal.NewFromFloat(10),
						},
					),
				)

				return nil
			},
		})
		s.NoError(err)

		s.Len(updatedInvoice.Discounts.OrEmpty(), 1)
		discount, err := updatedInvoice.Discounts.OrEmpty()[0].AsPercentage()
		s.NoError(err)

		s.Equal(discount.Name, "Test discount")
		s.Equal(discount.Percentage.InexactFloat64(), float64(10))
		s.Equal(discount.LineIDs, []string{lines[0].ID})
		s.Equal(discount.InvoiceID, gatheringInvoice.ID)
	})

	s.Run("Update discount", func() {
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: billing.InvoiceID{
				Namespace: namespace,
				ID:        lines[0].InvoiceID,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)

		originalDiscountID := lo.Must(gatheringInvoice.Discounts.OrEmpty()[0].DiscountBase()).ID

		updatedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: gatheringInvoice.InvoiceID(),
			EditFn: func(invoice *billing.Invoice) error {
				discounts := invoice.Discounts.OrEmpty()
				discount, err := discounts[0].AsPercentage()
				s.NoError(err)

				discount.Percentage = alpacadecimal.NewFromFloat(20)

				discounts[0] = billing.NewInvoiceDiscountFrom(discount)

				invoice.Discounts = billing.NewInvoiceDiscounts(discounts)
				return nil
			},
		})
		s.NoError(err)

		s.Len(updatedInvoice.Discounts.OrEmpty(), 1)
		discount, err := updatedInvoice.Discounts.OrEmpty()[0].AsPercentage()
		s.NoError(err)

		s.Equal(discount.Percentage.InexactFloat64(), float64(20))
		s.Equal(discount.ID, originalDiscountID)
	})

	s.Run("Delete discount", func() {
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: billing.InvoiceID{
				Namespace: namespace,
				ID:        lines[0].InvoiceID,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)

		existingDiscountID := lo.Must(gatheringInvoice.Discounts.OrEmpty()[0].DiscountBase()).DiscountID()

		updatedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: gatheringInvoice.InvoiceID(),
			EditFn: func(invoice *billing.Invoice) error {
				invoice.Discounts = billing.NewInvoiceDiscounts(nil)
				return nil
			},
		})
		s.NoError(err)

		s.True(updatedInvoice.Discounts.IsPresent())
		s.Len(updatedInvoice.Discounts.OrEmpty(), 0)

		discount, err := discountGetter.GetInvoiceDiscount(ctx, existingDiscountID)
		s.NoError(err)
		discountBase := lo.Must(discount.DiscountBase())
		s.NotNil(discountBase.DeletedAt)
	})

	s.Run("Create invalid invoice discount: line reference is invalid", func() {
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: billing.InvoiceID{
				Namespace: namespace,
				ID:        lines[0].InvoiceID,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)

		_, err = s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: gatheringInvoice.InvoiceID(),
			EditFn: func(invoice *billing.Invoice) error {
				invoice.Discounts.Append(
					billing.NewInvoiceDiscountFrom(
						billing.InvoiceDiscountPercentage{
							InvoiceDiscountBase: billing.InvoiceDiscountBase{
								ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
									Namespace: namespace,
									Name:      "Test discount",
								}),
								InvoiceID: gatheringInvoice.ID,
								Type:      billing.PercentageDiscountType,
								LineIDs:   []string{ulid.Make().String()},
							},
							Percentage: alpacadecimal.NewFromFloat(25),
						},
					),
				)

				return nil
			},
		})

		s.Error(err)
		s.ErrorIs(err, billing.ErrInvoiceDiscountInvalidLineReference)
		s.ErrorAs(err, &billing.ValidationError{})
	})

	s.Run("Create invalid invoice discount: wildcard discount on a gathering invoice", func() {
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: billing.InvoiceID{
				Namespace: namespace,
				ID:        lines[0].InvoiceID,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)

		_, err = s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: gatheringInvoice.InvoiceID(),
			EditFn: func(invoice *billing.Invoice) error {
				invoice.Discounts.Append(
					billing.NewInvoiceDiscountFrom(
						billing.InvoiceDiscountPercentage{
							InvoiceDiscountBase: billing.InvoiceDiscountBase{
								ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
									Namespace: namespace,
									Name:      "Test discount",
								}),
								InvoiceID: gatheringInvoice.ID,
								Type:      billing.PercentageDiscountType,
							},
							Percentage: alpacadecimal.NewFromFloat(25),
						},
					),
				)

				return nil
			},
		})

		s.Error(err)
		s.ErrorIs(err, billing.ErrInvoiceDiscountNoWildcardDiscountOnGatheringInvoices)
		s.ErrorAs(err, &billing.ValidationError{})
	})
}

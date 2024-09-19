package invoice_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/openmeterio/openmeter/openmeter/billing/invoice"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
)

type InvoicingTestSuite struct {
	BaseSuite
}

const (
	namespace = "default"
)

func TestInvoicing(t *testing.T) {
	suite.Run(t, new(InvoicingTestSuite))
}

func (s *InvoicingTestSuite) TestCreateInvoice() {
	// TODO: do we need this at all?
	clock.SetTime(testutils.GetRFC3339Time(s.T(), "2024-06-28T14:30:21Z"))
	defer clock.ResetTime()

	require := s.Require()
	// PgSQL has a time-resolution of 1microsecond
	now := time.Now().Truncate(time.Microsecond)
	periodEnd := now.Add(-time.Hour)
	periodStart := periodEnd.Add(-time.Hour * 24 * 30)
	issueAt := now.Add(-time.Minute)

	ctx := context.Background()

	// Let's create a test customer

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		Customer: customer.Customer{
			ManagedResource: models.ManagedResource{
				Key: "test-customer",
			},

			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			BillingAddress: &models.Address{
				Country:     lo.ToPtr(models.CountryCode("US")),
				PostalCode:  lo.ToPtr("12345"),
				State:       lo.ToPtr("NY"),
				City:        lo.ToPtr("New York"),
				Line1:       lo.ToPtr("1234 Test St"),
				Line2:       lo.ToPtr("Apt 1"),
				PhoneNumber: lo.ToPtr("1234567890"),
			},
		},
	})
	require.NoError(err)
	require.NotNil(customerEntity)
	require.NotEmpty(customerEntity.ID)

	// TODO: split into subtests
	var items []invoice.InvoiceItem

	s.T().Run("CreateInvoiceItems", func(t *testing.T) {
		// test create invoice
		items, err = s.InvoiceService.CreateInvoiceItems(ctx, nil, []invoice.InvoiceItem{
			{
				// TODO: this is crap like this :/
				ID: invoice.InvoiceItemID{
					Namespace: namespace,
				},
				CustomerID:  customerEntity.ID,
				PeriodStart: periodStart,
				PeriodEnd:   periodEnd,

				InvoiceAt: issueAt,

				Quantity:  alpacadecimal.NewFromFloat(1),
				UnitPrice: alpacadecimal.NewFromFloat(100),
				Currency:  currencyx.Code(currency.USD),

				Metadata: map[string]string{
					"key": "value",
				},
			},
			{
				ID: invoice.InvoiceItemID{
					Namespace: namespace,
				},
				CustomerID:  customerEntity.ID,
				PeriodStart: periodStart,
				PeriodEnd:   periodEnd,

				InvoiceAt: issueAt,

				Quantity:  alpacadecimal.NewFromFloat(3),
				UnitPrice: alpacadecimal.NewFromFloat(200),
				Currency:  currencyx.Code(currency.HUF),
			},
		})

		require.NoError(err)
		require.Len(items, 2)
		require.Equal(items[0], invoice.InvoiceItem{
			ID: invoice.InvoiceItemID{
				Namespace: namespace,
				ID:        items[0].ID.ID,
			},
			CustomerID: customerEntity.ID,

			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,

			InvoiceAt: issueAt,

			Quantity:  alpacadecimal.NewFromFloat(1),
			UnitPrice: alpacadecimal.NewFromFloat(100),
			Currency:  currencyx.Code(currency.USD),

			CreatedAt: items[0].CreatedAt,
			UpdatedAt: items[0].UpdatedAt,

			Metadata: map[string]string{
				"key": "value",
			},
		})
		require.NotEmpty(items[1].ID)
	})

	var pendingInvoice *invoice.Invoice
	s.T().Run("Pending invoices", func(t *testing.T) {
		// get pending items invoice
		pendingInvoice, err = s.InvoiceService.GetPendingInvoiceItems(ctx, customer.CustomerID{
			Namespace: namespace,
			ID:        customerEntity.ID,
		})

		require.NoError(err)
		require.NotNil(pendingInvoice)

		require.EqualValues(invoice.InvoiceCustomer{
			CustomerID:     customerEntity.ID,
			Name:           customerEntity.Name,
			BillingAddress: customerEntity.BillingAddress,
		}, pendingInvoice.Customer)

		require.Len(pendingInvoice.Items, 2)
		require.ElementsMatch(
			lo.Map(pendingInvoice.Items, func(item invoice.InvoiceItem, _ int) string {
				return item.ID.ID
			}),
			lo.Map(items, func(item invoice.InvoiceItem, _ int) string {
				return item.ID.ID
			}),
		)
	})

	testSupplier := invoice.InvoiceSupplier{
		Name:           "Test Supplier",
		TaxCountryCode: l10n.TaxCountryCode(l10n.US),
	}

	s.T().Run("Pending invoice - GOBL validation", func(t *testing.T) {
		pendingInvoiceGOBL, err := pendingInvoice.ToGOBL(invoice.GOBLMetadata{
			Supplier: testSupplier,
		})
		require.NoError(err)
		require.NotNil(pendingInvoiceGOBL)

		require.NoError(pendingInvoiceGOBL.Calculate(), "GOBL can calculate the invoice values")
		require.NoError(pendingInvoiceGOBL.Validate(), "GOBL can validate the invoice")
	})
}

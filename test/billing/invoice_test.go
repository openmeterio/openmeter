package billing_test

import (
	"context"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/gobldriver"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type InvoicingTestSuite struct {
	BaseSuite
}

func TestInvoicing(t *testing.T) {
	suite.Run(t, new(InvoicingTestSuite))
}

func (s *InvoicingTestSuite) TestPendingInvoiceValidation() {
	namespace := "ns-create-invoice-workflow"
	now := time.Now().Truncate(time.Microsecond)
	periodEnd := now.Add(-time.Hour)
	periodStart := periodEnd.Add(-time.Hour * 24 * 30)
	issueAt := now.Add(-time.Minute)

	ctx := context.Background()

	// Given we have a test customer

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: namespace,

		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: "Test Customer",
			}),
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
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	// Given we have a default profile for the namespace

	s.T().Run("create default profile", func(t *testing.T) {
		minimalCreateProfileInput := minimalCreateProfileInputTemplate
		minimalCreateProfileInput.Namespace = namespace

		profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

		require.NoError(t, err)
		require.NotNil(t, profile)
	})

	var items []billing.InvoiceItem
	var HUFItem billing.InvoiceItem

	s.T().Run("CreateInvoiceItems", func(t *testing.T) {
		// When we create invoice items

		items, err = s.BillingService.CreateInvoiceItems(ctx,
			billing.CreateInvoiceItemsInput{
				InvoiceID: nil,
				Namespace: namespace,
				Items: []billing.InvoiceItem{
					{
						Namespace:   namespace,
						CustomerID:  customerEntity.ID,
						PeriodStart: periodStart,
						PeriodEnd:   periodEnd,

						InvoiceAt: issueAt,

						Type: billing.InvoiceItemTypeStatic,

						Name:      "Test item - USD",
						Quantity:  lo.ToPtr(alpacadecimal.NewFromFloat(1)),
						UnitPrice: alpacadecimal.NewFromFloat(100),
						Currency:  currencyx.Code(currency.USD),

						Metadata: map[string]string{
							"key": "value",
						},
					},
					{
						Namespace:   namespace,
						CustomerID:  customerEntity.ID,
						PeriodStart: periodStart,
						PeriodEnd:   periodEnd,

						InvoiceAt: issueAt,

						Type: billing.InvoiceItemTypeStatic,

						Name:      "Test item - HUF",
						Quantity:  lo.ToPtr(alpacadecimal.NewFromFloat(3)),
						UnitPrice: alpacadecimal.NewFromFloat(200),
						Currency:  currencyx.Code(currency.HUF),
					},
				},
			})

		// Then we should have the items created
		require.NoError(s.T(), err)
		require.Len(s.T(), items, 2)
		require.Equal(s.T(), items[0], billing.InvoiceItem{
			ID:         items[0].ID,
			Namespace:  namespace,
			CustomerID: customerEntity.ID,

			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,

			InvoiceAt: issueAt,

			Type: billing.InvoiceItemTypeStatic,

			Name:      "Test item - USD",
			Quantity:  lo.ToPtr(alpacadecimal.NewFromFloat(1)),
			UnitPrice: alpacadecimal.NewFromFloat(100),
			Currency:  currencyx.Code(currency.USD),

			CreatedAt: items[0].CreatedAt,
			UpdatedAt: items[0].UpdatedAt,

			Metadata: map[string]string{
				"key": "value",
			},
		})
		require.NotEmpty(s.T(), items[1].ID)

		HUFItem = items[1]
	})

	var pendingInvoices []billing.InvoiceWithValidation
	var USDInvoice billing.InvoiceWithValidation

	s.T().Run("Pending invoices", func(t *testing.T) {
		// When we get the pending invoices
		pendingInvoices, err = s.BillingService.GetPendingInvoiceItems(ctx, customerentity.CustomerID{
			Namespace: namespace,
			ID:        customerEntity.ID,
		})

		// Then we should receive the one invoice per currency
		require.NoError(t, err)
		require.NotNil(t, pendingInvoices)
		require.Len(t, pendingInvoices, 2)

		sort.SliceStable(pendingInvoices, func(i, j int) bool {
			return pendingInvoices[i].Invoice.Currency < pendingInvoices[j].Invoice.Currency
		})

		for _, pendingInvoice := range pendingInvoices {
			pendingInvoice.Invoice.Customer.CreatedAt = customerEntity.CreatedAt
			pendingInvoice.Invoice.Customer.UpdatedAt = customerEntity.UpdatedAt

			require.EqualValues(t, billing.InvoiceCustomer(*customerEntity), pendingInvoice.Invoice.Customer)
		}

		require.EqualValues(t, HUFItem.ID, pendingInvoices[0].Invoice.Items[0].ID)

		USDInvoice = pendingInvoices[1]
	})

	s.T().Run("Pending invoice - GOBL validation", func(t *testing.T) {
		// When we validate the invoice with GOBL
		gobl, err := gobldriver.NewDriver(gobldriver.DriverConfig{
			Logger: slog.Default(),
		})

		// Then we should get no validation errors
		require.NoError(t, err)
		require.NotNil(t, gobl)

		invoice, err := gobl.Generate(context.Background(), USDInvoice)
		require.NoError(t, err)

		validationErrors, err := gobldriver.LookupValidationErrors(invoice)
		require.NoError(t, err)

		require.False(t, validationErrors.HasErrors(), "no validation errors expected")
	})
}

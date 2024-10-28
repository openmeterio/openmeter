package billing_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type InvoicingTestSuite struct {
	BaseSuite
}

func TestInvoicing(t *testing.T) {
	suite.Run(t, new(InvoicingTestSuite))
}

func (s *InvoicingTestSuite) TestPendingLineCreation() {
	namespace := "ns-create-invoice-workflow"
	now := time.Now().Truncate(time.Microsecond)
	periodEnd := now.Add(-time.Hour)
	periodStart := periodEnd.Add(-time.Hour * 24 * 30)
	issueAt := now.Add(-time.Minute)

	_ = s.installSandboxApp(s.T(), namespace)

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

	var billingProfile billingentity.Profile
	s.T().Run("create default profile", func(t *testing.T) {
		minimalCreateProfileInput := minimalCreateProfileInputTemplate
		minimalCreateProfileInput.Namespace = namespace

		profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

		require.NoError(t, err)
		require.NotNil(t, profile)
		billingProfile = *profile
	})

	var items []billingentity.Line
	var HUFItem billingentity.Line

	s.T().Run("CreateInvoiceItems", func(t *testing.T) {
		// When we create invoice items

		res, err := s.BillingService.CreateInvoiceLines(ctx,
			billing.CreateInvoiceLinesInput{
				Namespace:  namespace,
				CustomerID: customerEntity.ID,
				Lines: []billingentity.Line{
					{
						LineBase: billingentity.LineBase{
							Namespace: namespace,
							Period:    billingentity.Period{Start: periodStart, End: periodEnd},

							InvoiceAt: issueAt,

							Type: billingentity.InvoiceLineTypeManualFee,

							Name:     "Test item - USD",
							Currency: currencyx.Code(currency.USD),

							Metadata: map[string]string{
								"key": "value",
							},
						},
						ManualFee: &billingentity.ManualFeeLine{
							Price:    alpacadecimal.NewFromFloat(100),
							Quantity: alpacadecimal.NewFromFloat(1),
						},
					},
					{
						LineBase: billingentity.LineBase{
							Namespace: namespace,
							Period:    billingentity.Period{Start: periodStart, End: periodEnd},

							InvoiceAt: issueAt,

							Type: billingentity.InvoiceLineTypeManualFee,

							Name:     "Test item - HUF",
							Currency: currencyx.Code(currency.HUF),
						},
						ManualFee: &billingentity.ManualFeeLine{
							Price:    alpacadecimal.NewFromFloat(200),
							Quantity: alpacadecimal.NewFromFloat(3),
						},
					},
				},
			})

		// Then we should have the items created
		require.NoError(s.T(), err)
		items = res.Lines

		// Then we should have an usd invoice automatically created
		usdInvoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Page: pagination.Page{
				PageNumber: 1,
				PageSize:   10,
			},

			Namespace:  namespace,
			Customers:  []string{customerEntity.ID},
			Expand:     billing.InvoiceExpandAll,
			Statuses:   []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies: []currencyx.Code{currencyx.Code(currency.USD)},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), usdInvoices.Items, 1)
		usdInvoice := usdInvoices.Items[0]

		expectedUSDLine := billingentity.Line{
			LineBase: billingentity.LineBase{
				ID:        items[0].ID,
				Namespace: namespace,

				Period: billingentity.Period{Start: periodStart.Truncate(time.Microsecond), End: periodEnd.Truncate(time.Microsecond)},

				InvoiceID: usdInvoice.ID,
				InvoiceAt: issueAt,

				Type: billingentity.InvoiceLineTypeManualFee,

				Name:     "Test item - USD",
				Currency: currencyx.Code(currency.USD),

				CreatedAt: usdInvoice.Lines[0].CreatedAt,
				UpdatedAt: usdInvoice.Lines[0].UpdatedAt,

				Metadata: map[string]string{
					"key": "value",
				},
			},
			ManualFee: &billingentity.ManualFeeLine{
				Price:    alpacadecimal.NewFromFloat(100),
				Quantity: alpacadecimal.NewFromFloat(1),
			},
		}
		// Let's make sure that the workflow config is cloned
		require.NotEqual(s.T(), usdInvoice.Workflow.WorkflowConfig.ID, billingProfile.WorkflowConfig.ID)

		require.Equal(s.T(), usdInvoice, billingentity.Invoice{
			Namespace: namespace,
			ID:        usdInvoice.ID,

			Type:     billingentity.InvoiceTypeStandard,
			Currency: currencyx.Code(currency.USD),
			Status:   billingentity.InvoiceStatusGathering,

			CreatedAt: usdInvoice.CreatedAt,
			UpdatedAt: usdInvoice.UpdatedAt,

			Workflow: &billingentity.InvoiceWorkflow{
				WorkflowConfig: billingentity.WorkflowConfig{
					ID:        usdInvoice.Workflow.WorkflowConfig.ID,
					CreatedAt: usdInvoice.Workflow.WorkflowConfig.CreatedAt,
					UpdatedAt: usdInvoice.Workflow.WorkflowConfig.UpdatedAt,

					Timezone:   billingProfile.WorkflowConfig.Timezone,
					Collection: billingProfile.WorkflowConfig.Collection,
					Invoicing:  billingProfile.WorkflowConfig.Invoicing,
					Payment:    billingProfile.WorkflowConfig.Payment,
				},
				SourceBillingProfileID: billingProfile.ID,
				AppReferences:          *billingProfile.AppReferences,
				Apps:                   billingProfile.Apps,
			},

			Customer: billingentity.InvoiceCustomer{
				CustomerID: customerEntity.ID,

				Name:           customerEntity.Name,
				BillingAddress: customerEntity.BillingAddress,
			},
			Supplier: billingProfile.Supplier,

			Lines: []billingentity.Line{expectedUSDLine},
		})

		require.Len(s.T(), items, 2)
		// Validate that the create returns the expected items
		items[0].CreatedAt = expectedUSDLine.CreatedAt
		items[0].UpdatedAt = expectedUSDLine.UpdatedAt
		require.Equal(s.T(), items[0], expectedUSDLine)
		require.NotEmpty(s.T(), items[1].ID)

		HUFItem = items[1]
	})

	s.T().Run("CreateInvoiceItems - HUF", func(t *testing.T) {
		// Then a HUF item is also created
		require.NotNil(s.T(), HUFItem.ID)

		// Then we have a different invoice for HUF
		hufInvoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Page: pagination.Page{
				PageNumber: 1,
				PageSize:   10,
			},

			Namespace:  namespace,
			Customers:  []string{customerEntity.ID},
			Expand:     billing.InvoiceExpandAll,
			Statuses:   []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies: []currencyx.Code{currencyx.Code(currency.HUF)},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), hufInvoices.Items, 1)

		// Then we have one line item for the invoice
		require.Len(s.T(), hufInvoices.Items[0].Lines, 1)
	})

	s.T().Run("Expand scenarios - no expand", func(t *testing.T) {
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Page: pagination.Page{
				PageNumber: 1,
				PageSize:   10,
			},

			Namespace:  namespace,
			Customers:  []string{customerEntity.ID},
			Expand:     billing.InvoiceExpand{},
			Statuses:   []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies: []currencyx.Code{currencyx.Code(currency.USD)},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), invoices.Items, 1)
		invoice := invoices.Items[0]

		require.Len(s.T(), invoice.Lines, 0, "no lines should be returned")
		require.Nil(s.T(), invoice.Workflow, "no workflow should be returned")
	})

	s.T().Run("Expand scenarios - no app expand", func(t *testing.T) {
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Page: pagination.Page{
				PageNumber: 1,
				PageSize:   10,
			},

			Namespace: namespace,
			Customers: []string{customerEntity.ID},
			Expand: billing.InvoiceExpand{
				Workflow: true,
			},
			Statuses:   []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies: []currencyx.Code{currencyx.Code(currency.USD)},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), invoices.Items, 1)
		invoice := invoices.Items[0]

		require.Len(s.T(), invoice.Lines, 0, "no lines should be returned")
		require.NotNil(s.T(), invoice.Workflow, "workflow should be returned")
		require.Nil(s.T(), invoice.Workflow.Apps, "apps should not be resolved")
	})

	s.T().Run("Expand scenarios - app expand", func(t *testing.T) {
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Page: pagination.Page{
				PageNumber: 1,
				PageSize:   10,
			},

			Namespace: namespace,
			Customers: []string{customerEntity.ID},
			Expand: billing.InvoiceExpand{
				Workflow:     true,
				WorkflowApps: true,
			},
			Statuses:   []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies: []currencyx.Code{currencyx.Code(currency.USD)},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), invoices.Items, 1)
		invoice := invoices.Items[0]

		require.Len(s.T(), invoice.Lines, 0, "no lines should be returned")
		require.NotNil(s.T(), invoice.Workflow, "workflow should be returned")
		require.NotNil(s.T(), invoice.Workflow.Apps, "apps should  be resolved")
		require.NotNil(s.T(), invoice.Workflow.Apps.Tax, "apps should be resolved")
		require.NotNil(s.T(), invoice.Workflow.Apps.Invoicing, "apps should be resolved")
		require.NotNil(s.T(), invoice.Workflow.Apps.Payment, "apps should be resolved")
	})
}

func (s *InvoicingTestSuite) TestCreateInvoice() {
	namespace := "ns-create-invoice-gathering-to-draft"
	now := time.Now().Truncate(time.Microsecond)
	periodEnd := now.Add(-time.Hour)
	periodStart := periodEnd.Add(-time.Hour * 24 * 30)
	line1IssueAt := now.Add(-2 * time.Hour)
	line2IssueAt := now.Add(-time.Hour)

	_ = s.installSandboxApp(s.T(), namespace)

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
				Country: lo.ToPtr(models.CountryCode("US")),
			},
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	// Given we have a default profile for the namespace

	minimalCreateProfileInput := minimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = namespace

	profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

	require.NoError(s.T(), err)
	require.NotNil(s.T(), profile)

	res, err := s.BillingService.CreateInvoiceLines(ctx,
		billing.CreateInvoiceLinesInput{
			Namespace:  namespace,
			CustomerID: customerEntity.ID,
			Lines: []billingentity.Line{
				{
					LineBase: billingentity.LineBase{
						Namespace: namespace,
						Period:    billingentity.Period{Start: periodStart, End: periodEnd},

						InvoiceAt: line1IssueAt,

						Type: billingentity.InvoiceLineTypeManualFee,

						Name:     "Test item1",
						Currency: currencyx.Code(currency.USD),

						Metadata: map[string]string{
							"key": "value",
						},
					},
					ManualFee: &billingentity.ManualFeeLine{
						Price:    alpacadecimal.NewFromFloat(100),
						Quantity: alpacadecimal.NewFromFloat(1),
					},
				},
				{
					LineBase: billingentity.LineBase{
						Namespace: namespace,
						Period:    billingentity.Period{Start: periodStart, End: periodEnd},

						InvoiceAt: line2IssueAt,

						Type: billingentity.InvoiceLineTypeManualFee,

						Name:     "Test item2",
						Currency: currencyx.Code(currency.USD),
					},
					ManualFee: &billingentity.ManualFeeLine{
						Price:    alpacadecimal.NewFromFloat(200),
						Quantity: alpacadecimal.NewFromFloat(3),
					},
				},
			},
		})

	// Then we should have the items created
	require.NoError(s.T(), err)
	require.Len(s.T(), res.Lines, 2)
	line1ID := res.Lines[0].ID
	line2ID := res.Lines[1].ID
	require.NotEmpty(s.T(), line1ID)
	require.NotEmpty(s.T(), line2ID)

	// Expect that a single gathering invoice has been created
	require.Equal(s.T(), res.Lines[0].InvoiceID, res.Lines[1].InvoiceID)
	gatheringInvoiceID := billingentity.InvoiceID{
		Namespace: namespace,
		ID:        res.Lines[0].InvoiceID,
	}

	s.Run("Creating invoice in the future fails", func() {
		_, err := s.BillingService.CreateInvoiceAsOf(ctx, billing.CreateInvoiceAsOfInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			AsOf: lo.ToPtr(now.Add(time.Hour)),
		})

		require.Error(s.T(), err)
		require.ErrorAs(s.T(), err, &billing.ValidationError{})
	})

	s.Run("Creating invoice without any pending lines being available fails", func() {
		_, err := s.BillingService.CreateInvoiceAsOf(ctx, billing.CreateInvoiceAsOfInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},

			AsOf: lo.ToPtr(line1IssueAt.Add(-time.Minute)),
		})

		require.Error(s.T(), err)
		require.ErrorAs(s.T(), err, &billing.ValidationError{})
	})

	s.Run("Number of pending invoice lines is reported correctly by the adapter", func() {
		res, err := s.BillingAdapter.AssociatedLineCounts(ctx, billing.AssociatedLineCountsAdapterInput{
			Namespace:  namespace,
			InvoiceIDs: []string{gatheringInvoiceID.ID},
		})

		require.NoError(s.T(), err)
		require.Len(s.T(), res.Counts, 1)
		require.Equal(s.T(), int64(2), res.Counts[gatheringInvoiceID])
	})

	s.Run("When creating an invoice with only item1 included", func() {
		invoice, err := s.BillingService.CreateInvoiceAsOf(ctx, billing.CreateInvoiceAsOfInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			AsOf: lo.ToPtr(line1IssueAt.Add(time.Minute)),
		})

		// Then we should have the invoice created
		require.NoError(s.T(), err)
		require.Len(s.T(), invoice, 1)

		// Then we should have item1 added to the invoice
		require.Len(s.T(), invoice[0].Lines, 1)
		require.Equal(s.T(), line1ID, invoice[0].Lines[0].ID)

		// Then we expect that the gathering invoice is still present, with item2
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.InvoiceExpandAll,
		})
		require.NoError(s.T(), err)
		require.Nil(s.T(), gatheringInvoice.DeletedAt, "gathering invoice should be present")
		require.Len(s.T(), gatheringInvoice.Lines, 1)
		require.Equal(s.T(), line2ID, gatheringInvoice.Lines[0].ID)
	})

	s.Run("When creating an invoice with only item2 included, but bad asof", func() {
		_, err := s.BillingService.CreateInvoiceAsOf(ctx, billing.CreateInvoiceAsOfInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			IncludePendingLines: lo.ToPtr([]string{line2ID}),
			AsOf:                lo.ToPtr(line1IssueAt.Add(time.Minute)),
		})

		// Then we should receive a validation error
		require.Error(s.T(), err)
		require.ErrorAs(s.T(), err, &billing.ValidationError{})
	})

	s.Run("When creating an invoice with only item2 included", func() {
		invoice, err := s.BillingService.CreateInvoiceAsOf(ctx, billing.CreateInvoiceAsOfInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			IncludePendingLines: lo.ToPtr([]string{line2ID}),
			AsOf:                lo.ToPtr(now),
		})

		// Then we should have the invoice created
		require.NoError(s.T(), err)
		require.Len(s.T(), invoice, 1)

		// Then we should have item2 added to the invoice
		require.Len(s.T(), invoice[0].Lines, 1)
		require.Equal(s.T(), line2ID, invoice[0].Lines[0].ID)

		// Then we expect that the gathering invoice is deleted and empty
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.InvoiceExpandAll,
		})
		require.NoError(s.T(), err)
		require.NotNil(s.T(), gatheringInvoice.DeletedAt, "gathering invoice should be present")
		require.Len(s.T(), gatheringInvoice.Lines, 0, "deleted gathering invoice is empty")
	})

	s.Run("When include pending lines is an empty array", func() {
		invoice, err := s.BillingService.CreateInvoiceAsOf(ctx, billing.CreateInvoiceAsOfInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			IncludePendingLines: lo.ToPtr([]string{}),
		})

		// Then we should have the invoice created
		require.NoError(s.T(), err)
		// Without any items
		require.Len(s.T(), invoice, 0)
	})
}

package billing_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datex"
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

			Namespace:        namespace,
			Customers:        []string{customerEntity.ID},
			Expand:           billingentity.InvoiceExpandAll,
			ExtendedStatuses: []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies:       []currencyx.Code{currencyx.Code(currency.USD)},
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
		require.NotEqual(s.T(), usdInvoice.Workflow.Config.ID, billingProfile.WorkflowConfig.ID)

		require.Equal(s.T(), usdInvoice, billingentity.Invoice{
			Namespace: namespace,
			ID:        usdInvoice.ID,

			Type:          billingentity.InvoiceTypeStandard,
			Currency:      currencyx.Code(currency.USD),
			Status:        billingentity.InvoiceStatusGathering,
			StatusDetails: billingentity.InvoiceStatusDetails{},

			CreatedAt: usdInvoice.CreatedAt,
			UpdatedAt: usdInvoice.UpdatedAt,

			Workflow: &billingentity.InvoiceWorkflow{
				Config: billingentity.WorkflowConfig{
					ID:        usdInvoice.Workflow.Config.ID,
					CreatedAt: usdInvoice.Workflow.Config.CreatedAt,
					UpdatedAt: usdInvoice.Workflow.Config.UpdatedAt,

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

			ExpandedFields: billingentity.InvoiceExpandAll,
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

			Namespace:        namespace,
			Customers:        []string{customerEntity.ID},
			Expand:           billingentity.InvoiceExpandAll,
			ExtendedStatuses: []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies:       []currencyx.Code{currencyx.Code(currency.HUF)},
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

			Namespace:        namespace,
			Customers:        []string{customerEntity.ID},
			Expand:           billingentity.InvoiceExpand{},
			ExtendedStatuses: []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies:       []currencyx.Code{currencyx.Code(currency.USD)},
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
			Expand: billingentity.InvoiceExpand{
				Workflow: true,
			},
			ExtendedStatuses: []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies:       []currencyx.Code{currencyx.Code(currency.USD)},
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
			Expand: billingentity.InvoiceExpand{
				Workflow:     true,
				WorkflowApps: true,
			},
			ExtendedStatuses: []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies:       []currencyx.Code{currencyx.Code(currency.USD)},
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
		_, err := s.BillingService.CreateInvoice(ctx, billing.CreateInvoiceInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			AsOf: lo.ToPtr(now.Add(time.Hour)),
		})

		require.Error(s.T(), err)
		require.ErrorAs(s.T(), err, &billingentity.ValidationError{})
	})

	s.Run("Creating invoice without any pending lines being available fails", func() {
		_, err := s.BillingService.CreateInvoice(ctx, billing.CreateInvoiceInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},

			AsOf: lo.ToPtr(line1IssueAt.Add(-time.Minute)),
		})

		require.Error(s.T(), err)
		require.ErrorAs(s.T(), err, &billingentity.ValidationError{})
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
		invoice, err := s.BillingService.CreateInvoice(ctx, billing.CreateInvoiceInput{
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
			Expand:  billingentity.InvoiceExpandAll,
		})
		require.NoError(s.T(), err)
		require.Nil(s.T(), gatheringInvoice.DeletedAt, "gathering invoice should be present")
		require.Len(s.T(), gatheringInvoice.Lines, 1)
		require.Equal(s.T(), line2ID, gatheringInvoice.Lines[0].ID)
	})

	s.Run("When creating an invoice with only item2 included, but bad asof", func() {
		_, err := s.BillingService.CreateInvoice(ctx, billing.CreateInvoiceInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			IncludePendingLines: []string{line2ID},
			AsOf:                lo.ToPtr(line1IssueAt.Add(time.Minute)),
		})

		// Then we should receive a validation error
		require.Error(s.T(), err)
		require.ErrorAs(s.T(), err, &billingentity.ValidationError{})
	})

	s.Run("When creating an invoice with only item2 included", func() {
		invoice, err := s.BillingService.CreateInvoice(ctx, billing.CreateInvoiceInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			IncludePendingLines: []string{line2ID},
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
			Expand:  billingentity.InvoiceExpandAll,
		})
		require.NoError(s.T(), err)
		require.NotNil(s.T(), gatheringInvoice.DeletedAt, "gathering invoice should be present")
		require.Len(s.T(), gatheringInvoice.Lines, 0, "deleted gathering invoice is empty")
	})
}

type draftInvoiceInput struct {
	Namespace string
	Customer  *customerentity.Customer
}

func (i draftInvoiceInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := i.Customer.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *InvoicingTestSuite) createDraftInvoice(t *testing.T, ctx context.Context, in draftInvoiceInput) billingentity.Invoice {
	namespace := in.Customer.Namespace

	now := time.Now()
	invoiceAt := now.Add(-time.Second)
	periodEnd := now.Add(-24 * time.Hour)
	periodStart := periodEnd.Add(-24 * 30 * time.Hour)
	// Given we have a default profile for the namespace

	res, err := s.BillingService.CreateInvoiceLines(ctx,
		billing.CreateInvoiceLinesInput{
			Namespace:  in.Customer.Namespace,
			CustomerID: in.Customer.ID,
			Lines: []billingentity.Line{
				{
					LineBase: billingentity.LineBase{
						Namespace: namespace,
						Period:    billingentity.Period{Start: periodStart, End: periodEnd},

						InvoiceAt: invoiceAt,

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

						InvoiceAt: invoiceAt,

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

	require.NoError(s.T(), err)
	require.Len(s.T(), res.Lines, 2)
	line1ID := res.Lines[0].ID
	line2ID := res.Lines[1].ID
	require.NotEmpty(s.T(), line1ID)
	require.NotEmpty(s.T(), line2ID)

	invoice, err := s.BillingService.CreateInvoice(ctx, billing.CreateInvoiceInput{
		Customer: customerentity.CustomerID{
			ID:        in.Customer.ID,
			Namespace: in.Customer.Namespace,
		},
		AsOf: lo.ToPtr(now),
	})

	require.NoError(t, err)
	require.Len(t, invoice, 1)
	require.Len(t, invoice[0].Lines, 2)

	return invoice[0]
}

func (s *InvoicingTestSuite) TestInvoicingFlow() {
	cases := []struct {
		name           string
		workflowConfig billingentity.WorkflowConfig
		advance        func(t *testing.T, ctx context.Context, invoice billingentity.Invoice)
		expectedState  billingentity.InvoiceStatus
	}{
		{
			name: "instant issue",
			workflowConfig: billingentity.WorkflowConfig{
				Collection: billingentity.CollectionConfig{
					Alignment: billingentity.AlignmentKindSubscription,
				},
				Invoicing: billingentity.InvoicingConfig{
					AutoAdvance: true,
					DraftPeriod: lo.Must(datex.ISOString("PT0S").Parse()),
					DueAfter:    lo.Must(datex.ISOString("P1W").Parse()),
				},
				Payment: billingentity.PaymentConfig{
					CollectionMethod: billingentity.CollectionMethodChargeAutomatically,
				},
			},
			advance: func(t *testing.T, ctx context.Context, invoice billingentity.Invoice) {
				// When trying to advance an issued invoice, we get an error
				_, err := s.BillingService.AdvanceInvoice(ctx, billing.AdvanceInvoiceInput{
					ID:        invoice.ID,
					Namespace: invoice.Namespace,
				})

				require.ErrorIs(t, err, billingentity.ErrInvoiceCannotAdvance)
				require.ErrorAs(t, err, &billingentity.ValidationError{})
			},
			expectedState: billingentity.InvoiceStatusIssued,
		},
		{
			name: "draft period bypass with manual approve",
			workflowConfig: billingentity.WorkflowConfig{
				Collection: billingentity.CollectionConfig{
					Alignment: billingentity.AlignmentKindSubscription,
				},
				Invoicing: billingentity.InvoicingConfig{
					AutoAdvance: true,
					DraftPeriod: lo.Must(datex.ISOString("PT1H").Parse()),
					DueAfter:    lo.Must(datex.ISOString("P1W").Parse()),
				},
				Payment: billingentity.PaymentConfig{
					CollectionMethod: billingentity.CollectionMethodChargeAutomatically,
				},
			},
			advance: func(t *testing.T, ctx context.Context, invoice billingentity.Invoice) {
				require.Equal(s.T(), billingentity.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)

				// Approve the invoice, should become DraftReadyToIssue
				invoice, err := s.BillingService.ApproveInvoice(ctx, billing.ApproveInvoiceInput{
					ID:        invoice.ID,
					Namespace: invoice.Namespace,
				})

				require.NoError(s.T(), err)
				require.Equal(s.T(), billingentity.InvoiceStatusDraftReadyToIssue, invoice.Status)

				// Advance the invoice, should become Issued
				invoice, err = s.BillingService.AdvanceInvoice(ctx, billing.AdvanceInvoiceInput{
					ID:        invoice.ID,
					Namespace: invoice.Namespace,
				})

				require.NoError(s.T(), err)
				require.Equal(s.T(), billingentity.InvoiceStatusIssued, invoice.Status)
			},
			expectedState: billingentity.InvoiceStatusIssued,
		},
		{
			name: "manual approvement flow",
			workflowConfig: billingentity.WorkflowConfig{
				Collection: billingentity.CollectionConfig{
					Alignment: billingentity.AlignmentKindSubscription,
				},
				Invoicing: billingentity.InvoicingConfig{
					AutoAdvance: false,
					DraftPeriod: lo.Must(datex.ISOString("PT0H").Parse()),
					DueAfter:    lo.Must(datex.ISOString("P1W").Parse()),
				},
				Payment: billingentity.PaymentConfig{
					CollectionMethod: billingentity.CollectionMethodChargeAutomatically,
				},
			},
			advance: func(t *testing.T, ctx context.Context, invoice billingentity.Invoice) {
				require.Equal(s.T(), billingentity.InvoiceStatusDraftManualApprovalNeeded, invoice.Status)
				require.Equal(s.T(), billingentity.InvoiceStatusDetails{
					AvailableActions: []billingentity.InvoiceAction{billingentity.InvoiceActionApprove},
				}, invoice.StatusDetails)

				// Approve the invoice, should become DraftReadyToIssue
				invoice, err := s.BillingService.ApproveInvoice(ctx, billing.ApproveInvoiceInput{
					ID:        invoice.ID,
					Namespace: invoice.Namespace,
				})

				require.NoError(s.T(), err)
				require.Equal(s.T(), billingentity.InvoiceStatusDraftReadyToIssue, invoice.Status)

				// Advance the invoice, should become Issued
				invoice, err = s.BillingService.AdvanceInvoice(ctx, billing.AdvanceInvoiceInput{
					ID:        invoice.ID,
					Namespace: invoice.Namespace,
				})

				require.NoError(s.T(), err)
				require.Equal(s.T(), billingentity.InvoiceStatusIssued, invoice.Status)
			},
			expectedState: billingentity.InvoiceStatusIssued,
		},
	}

	ctx := context.Background()

	for i, tc := range cases {
		s.T().Run(tc.name, func(t *testing.T) {
			namespace := fmt.Sprintf("ns-invoicing-flow-happy-path-%d", i)

			_ = s.installSandboxApp(s.T(), namespace)

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

			// Given we have a billing profile
			minimalCreateProfileInput := minimalCreateProfileInputTemplate
			minimalCreateProfileInput.Namespace = namespace
			minimalCreateProfileInput.WorkflowConfig = tc.workflowConfig

			profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

			require.NoError(s.T(), err)
			require.NotNil(s.T(), profile)

			invoice := s.createDraftInvoice(s.T(), ctx, draftInvoiceInput{
				Namespace: namespace,
				Customer:  customerEntity,
			})
			require.NotNil(s.T(), invoice)

			// When we advance the invoice
			tc.advance(t, ctx, invoice)

			resultingInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: billingentity.InvoiceID{
					Namespace: namespace,
					ID:        invoice.ID,
				},
				Expand: billingentity.InvoiceExpandAll,
			})

			require.NoError(s.T(), err)
			require.NotNil(s.T(), resultingInvoice)
			require.Equal(s.T(), tc.expectedState, resultingInvoice.Status)
		})
	}
}

type ValidationIssueIntrospector interface {
	IntrospectValidationIssues(ctx context.Context, invoice billingentity.InvoiceID) ([]billingadapter.ValidationIssueWithDBMeta, error)
}

func (s *InvoicingTestSuite) TestInvoicingFlowErrorHandling() {
	cases := []struct {
		name           string
		workflowConfig billingentity.WorkflowConfig
		advance        func(t *testing.T, ctx context.Context, ns string, customer *customerentity.Customer, mockApp *appsandbox.MockApp) *billingentity.Invoice
		expectedState  billingentity.InvoiceStatus
	}{
		{
			name: "validation issue - different sources",
			workflowConfig: billingentity.WorkflowConfig{
				Collection: billingentity.CollectionConfig{
					Alignment: billingentity.AlignmentKindSubscription,
				},
				Invoicing: billingentity.InvoicingConfig{
					AutoAdvance: true,
					DraftPeriod: lo.Must(datex.ISOString("PT0S").Parse()),
					DueAfter:    lo.Must(datex.ISOString("P1W").Parse()),
				},
				Payment: billingentity.PaymentConfig{
					CollectionMethod: billingentity.CollectionMethodChargeAutomatically,
				},
			},
			advance: func(t *testing.T, ctx context.Context, ns string, customer *customerentity.Customer, mockApp *appsandbox.MockApp) *billingentity.Invoice {
				calcMock := s.InvoiceCalculator.EnableMock()
				defer s.InvoiceCalculator.DisableMock()

				validationIssueGetter, ok := s.BillingAdapter.(ValidationIssueIntrospector)
				require.True(t, ok)

				// Given that the app will return a validation error
				mockApp.On("ValidateInvoice", mock.Anything, mock.Anything).
					Return(billingentity.NewValidationError("test1", "validation error")).Once()
				calcMock.On("Calculate", mock.Anything).
					Return(nil).Once()

				// When we create a draft invoice
				invoice := s.createDraftInvoice(s.T(), ctx, draftInvoiceInput{
					Namespace: ns,
					Customer:  customer,
				})
				require.NotNil(s.T(), invoice)

				// Then we should end up in draft_invalid state
				require.Equal(s.T(), billingentity.InvoiceStatusDraftInvalid, invoice.Status)
				require.Equal(s.T(), billingentity.InvoiceStatusDetails{
					AvailableActions: []billingentity.InvoiceAction{
						billingentity.InvoiceActionRetry,
					},
					Immutable: false,
				}, invoice.StatusDetails)
				require.Equal(s.T(), billingentity.ValidationIssues{
					{
						Severity:  billingentity.ValidationIssueSeverityCritical,
						Code:      "test1",
						Message:   "validation error",
						Component: "app/sandbox/invoiceCustomers",
					},
				}, invoice.ValidationIssues)

				// Then we have the issues captured in the database
				issues, err := validationIssueGetter.IntrospectValidationIssues(ctx, billingentity.InvoiceID{
					Namespace: ns,
					ID:        invoice.ID,
				})
				require.NoError(t, err)
				require.Len(t, issues, 1)
				require.Equal(t,
					billingentity.ValidationIssue{
						Severity:  billingentity.ValidationIssueSeverityCritical,
						Code:      "test1",
						Message:   "validation error",
						Component: "app/sandbox/invoiceCustomers",
					},
					issues[0].ValidationIssue,
				)
				require.Nil(t, issues[0].DeletedAt)
				customerValidationIssueID := issues[0].ID
				require.NotEmpty(t, customerValidationIssueID)

				// Given that the issue is fixed, but a new one is introduced by editing the invoice
				mockApp.On("ValidateInvoice", mock.Anything, mock.Anything).
					Return(nil).Once()
				calcMock.On("Calculate", mock.Anything).
					Return(billingentity.NewValidationError("test2", "validation error")).Once()

				// TODO: we should trigger the update of the invoice here, but that's not yet available
				// regardless the state transition will be the same for now.
				invoice, err = s.BillingService.RetryInvoice(ctx, billing.RetryInvoiceInput{
					ID:        invoice.ID,
					Namespace: invoice.Namespace,
				})
				require.NoError(s.T(), err)
				require.NotNil(s.T(), invoice)

				// Then we should end up in draft_invalid state
				require.Equal(s.T(), billingentity.InvoiceStatusDraftInvalid, invoice.Status)
				require.Equal(s.T(), billingentity.InvoiceStatusDetails{
					AvailableActions: []billingentity.InvoiceAction{
						billingentity.InvoiceActionRetry,
					},
					Immutable: false,
				}, invoice.StatusDetails)
				require.Equal(s.T(), billingentity.ValidationIssues{
					{
						Severity:  billingentity.ValidationIssueSeverityCritical,
						Code:      "test2",
						Message:   "validation error",
						Component: billingentity.ValidationComponentOpenMeter,
					},
				}, invoice.ValidationIssues)

				// Then we have the new issues captured in the database, the old one deleted
				issues, err = validationIssueGetter.IntrospectValidationIssues(ctx, billingentity.InvoiceID{
					Namespace: ns,
					ID:        invoice.ID,
				})
				require.NoError(t, err)
				require.Len(t, issues, 2)

				// The old issue should be deleted
				invoiceIssue, ok := lo.Find(issues, func(i billingadapter.ValidationIssueWithDBMeta) bool {
					return i.ID == customerValidationIssueID
				})
				require.True(t, ok, "old issue should be present")
				require.NotNil(t, invoiceIssue.DeletedAt)
				require.Equal(t,
					billingentity.ValidationIssue{
						Severity:  billingentity.ValidationIssueSeverityCritical,
						Code:      "test1",
						Message:   "validation error",
						Component: "app/sandbox/invoiceCustomers",
					},
					invoiceIssue.ValidationIssue,
				)

				// The new issue should not be deleted
				calculationErrorIssue, ok := lo.Find(issues, func(i billingadapter.ValidationIssueWithDBMeta) bool {
					return i.ID != customerValidationIssueID
				})
				require.True(t, ok, "new issue should be present")
				require.Nil(t, calculationErrorIssue.DeletedAt)
				require.Equal(t,
					billingentity.ValidationIssue{
						Severity:  billingentity.ValidationIssueSeverityCritical,
						Code:      "test2",
						Message:   "validation error",
						Component: "openmeter",
					},
					calculationErrorIssue.ValidationIssue,
				)

				// TODO: validate db storage of validation issues too
				// Given that both issues are present, both will be reported
				mockApp.On("ValidateInvoice", mock.Anything, mock.Anything).
					Return(billingentity.NewValidationError("test1", "validation error")).Once()
				calcMock.On("Calculate", mock.Anything).
					Return(billingentity.NewValidationError("test2", "validation error")).Once()

				// TODO: we should trigger the update of the invoice here, but that's not yet available
				// regardless the state transition will be the same for now.
				invoice, err = s.BillingService.RetryInvoice(ctx, billing.RetryInvoiceInput{
					ID:        invoice.ID,
					Namespace: invoice.Namespace,
				})
				require.NoError(s.T(), err)
				require.NotNil(s.T(), invoice)

				// Then we should end up in draft_invalid state
				require.Equal(s.T(), billingentity.InvoiceStatusDraftInvalid, invoice.Status)
				require.Equal(s.T(), billingentity.InvoiceStatusDetails{
					AvailableActions: []billingentity.InvoiceAction{
						billingentity.InvoiceActionRetry,
					},
					Immutable: false,
				}, invoice.StatusDetails)
				require.ElementsMatch(s.T(), billingentity.ValidationIssues{
					{
						Severity:  billingentity.ValidationIssueSeverityCritical,
						Code:      "test1",
						Message:   "validation error",
						Component: "app/sandbox/invoiceCustomers",
					},
					{
						Severity:  billingentity.ValidationIssueSeverityCritical,
						Code:      "test2",
						Message:   "validation error",
						Component: billingentity.ValidationComponentOpenMeter,
					},
				}, invoice.ValidationIssues)

				// The database now has both issues active (but no new ones are created)
				issues, err = validationIssueGetter.IntrospectValidationIssues(ctx, billingentity.InvoiceID{
					Namespace: ns,
					ID:        invoice.ID,
				})
				require.NoError(t, err)
				require.Len(t, issues, 2)

				_, deletedIssueFound := lo.Find(issues, func(i billingadapter.ValidationIssueWithDBMeta) bool {
					return i.DeletedAt != nil
				})
				require.False(t, deletedIssueFound, "no issues should be deleted")

				return &invoice
			},
			expectedState: billingentity.InvoiceStatusDraftInvalid,
		},
		{
			name: "validation issue - warnings allow state transitions",
			workflowConfig: billingentity.WorkflowConfig{
				Collection: billingentity.CollectionConfig{
					Alignment: billingentity.AlignmentKindSubscription,
				},
				Invoicing: billingentity.InvoicingConfig{
					AutoAdvance: true,
					DraftPeriod: lo.Must(datex.ISOString("PT0S").Parse()),
					DueAfter:    lo.Must(datex.ISOString("P1W").Parse()),
				},
				Payment: billingentity.PaymentConfig{
					CollectionMethod: billingentity.CollectionMethodChargeAutomatically,
				},
			},
			advance: func(t *testing.T, ctx context.Context, ns string, customer *customerentity.Customer, mockApp *appsandbox.MockApp) *billingentity.Invoice {
				calcMock := s.InvoiceCalculator.EnableMock()
				defer s.InvoiceCalculator.DisableMock()

				// Given that the app will return a validation error
				mockApp.On("ValidateInvoice", mock.Anything, mock.Anything).
					Return(billingentity.NewValidationWarning("test1", "validation warning")).Once()
				calcMock.On("Calculate", mock.Anything).
					Return(nil).Once()

				// When we create a draft invoice
				invoice := s.createDraftInvoice(s.T(), ctx, draftInvoiceInput{
					Namespace: ns,
					Customer:  customer,
				})
				require.NotNil(s.T(), invoice)

				// Then we should end up in draft_invalid state
				require.Equal(s.T(), billingentity.InvoiceStatusIssued, invoice.Status)
				require.Equal(s.T(), billingentity.InvoiceStatusDetails{
					AvailableActions: []billingentity.InvoiceAction{},
					Immutable:        true,
				}, invoice.StatusDetails)
				require.Equal(s.T(), billingentity.ValidationIssues{
					{
						Severity:  billingentity.ValidationIssueSeverityWarning,
						Code:      "test1",
						Message:   "validation warning",
						Component: "app/sandbox/invoiceCustomers",
					},
				}, invoice.ValidationIssues)

				return &invoice
			},
			expectedState: billingentity.InvoiceStatusIssued,
		},
	}

	ctx := context.Background()

	for i, tc := range cases {
		s.T().Run(tc.name, func(t *testing.T) {
			namespace := fmt.Sprintf("ns-invoicing-flow-valid-%d", i)

			_ = s.installSandboxApp(s.T(), namespace)

			mockApp := s.SandboxApp.EnableMock(t)
			defer s.SandboxApp.DisableMock()

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

			// Given we have a billing profile
			minimalCreateProfileInput := minimalCreateProfileInputTemplate
			minimalCreateProfileInput.Namespace = namespace
			minimalCreateProfileInput.WorkflowConfig = tc.workflowConfig

			profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

			require.NoError(s.T(), err)
			require.NotNil(s.T(), profile)

			// When we advance the invoice
			invoice := tc.advance(t, ctx, namespace, customerEntity, mockApp)

			resultingInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: billingentity.InvoiceID{
					Namespace: namespace,
					ID:        invoice.ID,
				},
				Expand: billingentity.InvoiceExpandAll,
			})

			require.NoError(s.T(), err)
			require.NotNil(s.T(), resultingInvoice)
			require.Equal(s.T(), tc.expectedState, resultingInvoice.Status)
		})
	}
}

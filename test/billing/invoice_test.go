package billing_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
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
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)
	periodEnd := now.Add(-time.Hour)
	periodStart := periodEnd.Add(-time.Hour * 24 * 30)
	issueAt := now.Add(-time.Minute)

	_ = s.installSandboxApp(s.T(), namespace)

	ctx := context.Background()

	// Given we have a test customer

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customerentity.CustomerMutate{
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
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	s.MeterRepo.ReplaceMeters(ctx, []models.Meter{
		{
			Namespace:   namespace,
			Slug:        "test",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
	})
	defer s.MeterRepo.ReplaceMeters(ctx, []models.Meter{})

	_, err = s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      "test",
		Key:       "test",
		MeterSlug: lo.ToPtr("test"),
	})
	require.NoError(s.T(), err)

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

	var items []*billingentity.Line
	var HUFItem *billingentity.Line

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

							Type: billingentity.InvoiceLineTypeFee,

							Name:     "Test item - USD",
							Currency: currencyx.Code(currency.USD),

							Metadata: map[string]string{
								"key": "value",
							},
						},
						FlatFee: billingentity.FlatFeeLine{
							PerUnitAmount: alpacadecimal.NewFromFloat(100),
							Quantity:      alpacadecimal.NewFromFloat(1),
						},
					},
					{
						LineBase: billingentity.LineBase{
							Period: billingentity.Period{Start: periodStart, End: periodEnd},

							InvoiceAt: issueAt,

							Type: billingentity.InvoiceLineTypeFee,

							Name:     "Test item - HUF",
							Currency: currencyx.Code(currency.HUF),
						},
						FlatFee: billingentity.FlatFeeLine{
							PerUnitAmount: alpacadecimal.NewFromFloat(200),
							Quantity:      alpacadecimal.NewFromFloat(3),
						},
					},
					{
						LineBase: billingentity.LineBase{
							Period: billingentity.Period{Start: periodStart, End: periodEnd},

							InvoiceAt: issueAt,

							Type: billingentity.InvoiceLineTypeUsageBased,

							Name:     "Test item - HUF",
							Currency: currencyx.Code(currency.HUF),
						},
						UsageBased: billingentity.UsageBasedLine{
							Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
								Mode: productcatalog.GraduatedTieredPrice,
								Tiers: []productcatalog.PriceTier{
									{
										UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(10),
										},
									},
									{
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(100),
										},
									},
								},
							}),
							FeatureKey: "test",
						},
					},
				},
			})

		// Then we should have the items created
		require.NoError(s.T(), err)
		items = res

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

		usdInvoiceLine := usdInvoice.Lines.MustGet()[0]
		expectedUSDLine := &billingentity.Line{
			LineBase: billingentity.LineBase{
				ID:        items[0].ID,
				Namespace: namespace,

				Period: billingentity.Period{Start: periodStart.Truncate(time.Microsecond), End: periodEnd.Truncate(time.Microsecond)},

				InvoiceID: usdInvoice.ID,
				InvoiceAt: issueAt.In(time.UTC),

				Type: billingentity.InvoiceLineTypeFee,

				Name:     "Test item - USD",
				Currency: currencyx.Code(currency.USD),

				Status: billingentity.InvoiceLineStatusValid,

				CreatedAt: usdInvoiceLine.CreatedAt.In(time.UTC),
				UpdatedAt: usdInvoiceLine.UpdatedAt.In(time.UTC),

				Metadata: map[string]string{
					"key": "value",
				},
			},
			FlatFee: billingentity.FlatFeeLine{
				ConfigID:      usdInvoiceLine.FlatFee.ConfigID,
				PerUnitAmount: alpacadecimal.NewFromFloat(100),
				Quantity:      alpacadecimal.NewFromFloat(1),
			},
		}
		// Let's make sure that the workflow config is cloned
		require.NotEqual(s.T(), usdInvoice.Workflow.Config.ID, billingProfile.WorkflowConfig.ID)
		expectedInvoice := billingentity.Invoice{
			InvoiceBase: billingentity.InvoiceBase{
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
					UsageAttribution: billingentity.CustomerUsageAttribution{
						SubjectKeys: []string{"test"},
					},
				},
				Supplier: billingProfile.Supplier,
			},

			Lines: billingentity.NewLineChildren([]*billingentity.Line{expectedUSDLine}),

			ExpandedFields: billingentity.InvoiceExpandAll,
		}

		require.Equal(s.T(),
			expectedInvoice.RemoveMetaForCompare(),
			usdInvoice.RemoveMetaForCompare())

		require.Len(s.T(), items, 3)
		// Validate that the create returns the expected items
		items[0].CreatedAt = expectedUSDLine.CreatedAt
		items[0].UpdatedAt = expectedUSDLine.UpdatedAt
		require.Equal(s.T(), items[0].RemoveMetaForCompare(), expectedUSDLine.RemoveMetaForCompare())
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

		hufInvoiceLines := hufInvoices.Items[0].Lines.MustGet()

		// Then we have two line items for the invoice
		require.Len(s.T(), hufInvoiceLines, 2)

		_, found := lo.Find(hufInvoiceLines, func(l *billingentity.Line) bool {
			return l.Type == billingentity.InvoiceLineTypeFee
		})
		require.True(s.T(), found, "manual fee item is present")

		// Then we should have the tiered price present
		tieredLine, found := lo.Find(hufInvoiceLines, func(l *billingentity.Line) bool {
			return l.Type == billingentity.InvoiceLineTypeUsageBased
		})

		require.True(s.T(), found, "tiered price item is present")
		require.Equal(s.T(), tieredLine.UsageBased.Price.Type(), productcatalog.TieredPriceType)
		tieredPrice, err := tieredLine.UsageBased.Price.AsTiered()
		require.NoError(s.T(), err)

		require.Equal(s.T(),
			tieredPrice,
			productcatalog.TieredPrice{
				Mode: productcatalog.GraduatedTieredPrice,
				Tiers: []productcatalog.PriceTier{
					{
						UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
						UnitPrice: &productcatalog.PriceTierUnitPrice{
							Amount: alpacadecimal.NewFromFloat(10),
						},
					},
					{
						UnitPrice: &productcatalog.PriceTierUnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						},
					},
				},
			},
		)
	})

	s.T().Run("Expand scenarios - no  expand", func(t *testing.T) {
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

		require.False(s.T(), invoice.Lines.IsPresent(), "no lines should be returned")
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
				WorkflowApps: true,
			},
			ExtendedStatuses: []billingentity.InvoiceStatus{billingentity.InvoiceStatusGathering},
			Currencies:       []currencyx.Code{currencyx.Code(currency.USD)},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), invoices.Items, 1)
		invoice := invoices.Items[0]

		require.False(s.T(), invoice.Lines.IsPresent(), "no lines should be returned")
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

		CustomerMutate: customerentity.CustomerMutate{
			Name:         "Test Customer",
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

						Type: billingentity.InvoiceLineTypeFee,

						Name:     "Test item1",
						Currency: currencyx.Code(currency.USD),

						Metadata: map[string]string{
							"key": "value",
						},
					},
					FlatFee: billingentity.FlatFeeLine{
						PerUnitAmount: alpacadecimal.NewFromFloat(100),
						Quantity:      alpacadecimal.NewFromFloat(1),
					},
				},
				{
					LineBase: billingentity.LineBase{
						Namespace: namespace,
						Period:    billingentity.Period{Start: periodStart, End: periodEnd},

						InvoiceAt: line2IssueAt,

						Type: billingentity.InvoiceLineTypeFee,

						Name:     "Test item2",
						Currency: currencyx.Code(currency.USD),
					},
					FlatFee: billingentity.FlatFeeLine{
						PerUnitAmount: alpacadecimal.NewFromFloat(200),
						Quantity:      alpacadecimal.NewFromFloat(3),
					},
				},
			},
		})

	// Then we should have the items created
	require.NoError(s.T(), err)
	require.Len(s.T(), res, 2)
	line1ID := res[0].ID
	line2ID := res[1].ID
	require.NotEmpty(s.T(), line1ID)
	require.NotEmpty(s.T(), line2ID)

	// Expect that a single gathering invoice has been created
	require.Equal(s.T(), res[0].InvoiceID, res[1].InvoiceID)
	gatheringInvoiceID := billingentity.InvoiceID{
		Namespace: namespace,
		ID:        res[0].InvoiceID,
	}

	s.Run("Creating invoice in the future fails", func() {
		_, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
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
		_, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
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
		invoice, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
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
		require.Len(s.T(), invoice[0].Lines.MustGet(), 1)
		require.Equal(s.T(), line1ID, invoice[0].Lines.MustGet()[0].ID)

		// Then we expect that the gathering invoice is still present, with item2
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billingentity.InvoiceExpandAll,
		})
		require.NoError(s.T(), err)
		require.Nil(s.T(), gatheringInvoice.DeletedAt, "gathering invoice should be present")
		require.Len(s.T(), gatheringInvoice.Lines.MustGet(), 1)
		require.Equal(s.T(), line2ID, gatheringInvoice.Lines.MustGet()[0].ID)
	})

	s.Run("When creating an invoice with only item2 included, but bad asof", func() {
		_, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			IncludePendingLines: mo.Some([]string{line2ID}),
			AsOf:                lo.ToPtr(line1IssueAt.Add(time.Minute)),
		})

		// Then we should receive a validation error
		require.Error(s.T(), err)
		require.ErrorAs(s.T(), err, &billingentity.ValidationError{})
	})

	s.Run("When creating an invoice with only item2 included", func() {
		invoice, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
			IncludePendingLines: mo.Some([]string{line2ID}),
			AsOf:                lo.ToPtr(now),
		})

		// Then we should have the invoice created
		require.NoError(s.T(), err)
		require.Len(s.T(), invoice, 1)

		// Then we should have item2 added to the invoice
		require.Len(s.T(), invoice[0].Lines.MustGet(), 1)
		require.Equal(s.T(), line2ID, invoice[0].Lines.MustGet()[0].ID)

		// Then we expect that the gathering invoice is deleted and empty
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billingentity.InvoiceExpandAll,
		})
		require.NoError(s.T(), err)
		require.NotNil(s.T(), gatheringInvoice.DeletedAt, "gathering invoice should be present")
		require.Len(s.T(), gatheringInvoice.Lines.MustGet(), 0, "deleted gathering invoice is empty")
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

						Type: billingentity.InvoiceLineTypeFee,

						Name:     "Test item1",
						Currency: currencyx.Code(currency.USD),

						Metadata: map[string]string{
							"key": "value",
						},
					},
					FlatFee: billingentity.FlatFeeLine{
						PerUnitAmount: alpacadecimal.NewFromFloat(100),
						Quantity:      alpacadecimal.NewFromFloat(1),
					},
				},
				{
					LineBase: billingentity.LineBase{
						Namespace: namespace,
						Period:    billingentity.Period{Start: periodStart, End: periodEnd},

						InvoiceAt: invoiceAt,

						Type: billingentity.InvoiceLineTypeFee,

						Name:     "Test item2",
						Currency: currencyx.Code(currency.USD),
					},
					FlatFee: billingentity.FlatFeeLine{
						PerUnitAmount: alpacadecimal.NewFromFloat(200),
						Quantity:      alpacadecimal.NewFromFloat(3),
					},
				},
			},
		})

	require.NoError(s.T(), err)
	require.Len(s.T(), res, 2)
	line1ID := res[0].ID
	line2ID := res[1].ID
	require.NotEmpty(s.T(), line1ID)
	require.NotEmpty(s.T(), line2ID)

	invoice, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerentity.CustomerID{
			ID:        in.Customer.ID,
			Namespace: in.Customer.Namespace,
		},
		AsOf: lo.ToPtr(now),
	})

	require.NoError(t, err)
	require.Len(t, invoice, 1)
	require.Len(t, invoice[0].Lines.MustGet(), 2)

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

				// Approve the invoice, should become Issued
				invoice, err := s.BillingService.ApproveInvoice(ctx, billing.ApproveInvoiceInput{
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

				CustomerMutate: customerentity.CustomerMutate{
					Name:         "Test Customer",
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
				defer s.InvoiceCalculator.DisableMock(t)

				validationIssueGetter, ok := s.BillingAdapter.(ValidationIssueIntrospector)
				require.True(t, ok)

				// Given that the app will return a validation error
				mockApp.OnValidateInvoice(billingentity.NewValidationError("test1", "validation error"))
				calcMock.OnCalculate(nil)

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
						Component: "app.sandbox.invoiceCustomers.validate",
					},
				}, invoice.ValidationIssues.RemoveMetaForCompare())

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
						Component: "app.sandbox.invoiceCustomers.validate",
					},
					issues[0].ValidationIssue,
				)
				require.Nil(t, issues[0].DeletedAt)
				customerValidationIssueID := issues[0].ID
				require.NotEmpty(t, customerValidationIssueID)

				calcMock.AssertExpectations(t)
				mockApp.Reset(t)

				// Given that the issue is fixed, but a new one is introduced by editing the invoice
				mockApp.OnValidateInvoice(nil)
				calcMock.OnCalculate(billingentity.NewValidationError("test2", "validation error"))

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
				}, invoice.ValidationIssues.RemoveMetaForCompare())

				// Then we have the new issues captured in the database, the old one deleted, as Retry changes the severity
				// we will have a new validation issue
				issues, err = validationIssueGetter.IntrospectValidationIssues(ctx, billingentity.InvoiceID{
					Namespace: ns,
					ID:        invoice.ID,
				})
				require.NoError(t, err)
				require.Len(t, issues, 3)

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
						Component: "app.sandbox.invoiceCustomers.validate",
					},
					invoiceIssue.ValidationIssue,
				)

				// A new version of the issue is present with downgraded severity, to facilitate the retry
				downgradedIssue, ok := lo.Find(issues, func(i billingadapter.ValidationIssueWithDBMeta) bool {
					return i.Code == "test1" && i.Severity == billingentity.ValidationIssueSeverityWarning
				})
				require.True(t, ok, "the issue should be present")
				require.NotNil(t, downgradedIssue.DeletedAt)
				require.Equal(t,
					billingentity.ValidationIssue{
						Severity:  billingentity.ValidationIssueSeverityWarning,
						Code:      "test1",
						Message:   "validation error",
						Component: "app.sandbox.invoiceCustomers.validate",
					},
					downgradedIssue.ValidationIssue,
				)

				// The new issue should not be deleted
				calculationErrorIssue, ok := lo.Find(issues, func(i billingadapter.ValidationIssueWithDBMeta) bool {
					return i.Code == "test2"
				})
				require.True(t, ok, "new issue should be present")
				require.Equal(t,
					billingentity.ValidationIssue{
						Severity:  billingentity.ValidationIssueSeverityCritical,
						Code:      "test2",
						Message:   "validation error",
						Component: "openmeter",
					},
					calculationErrorIssue.ValidationIssue,
				)

				mockApp.Reset(t)
				calcMock.Reset(t)

				// Given that both issues are present, both will be reported
				mockApp.OnValidateInvoice(billingentity.NewValidationError("test1", "validation error"))
				calcMock.OnCalculate(billingentity.NewValidationError("test2", "validation error"))

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
						Component: "app.sandbox.invoiceCustomers.validate",
					},
					{
						Severity:  billingentity.ValidationIssueSeverityCritical,
						Code:      "test2",
						Message:   "validation error",
						Component: billingentity.ValidationComponentOpenMeter,
					},
				}, invoice.ValidationIssues.RemoveMetaForCompare())

				// The database now has both  critical issues active (but no new ones are created)
				issues, err = validationIssueGetter.IntrospectValidationIssues(ctx, billingentity.InvoiceID{
					Namespace: ns,
					ID:        invoice.ID,
				})
				require.NoError(t, err)
				criticalIssues := lo.Filter(issues, func(i billingadapter.ValidationIssueWithDBMeta, _ int) bool {
					return i.Severity == billingentity.ValidationIssueSeverityCritical
				})
				require.Len(t, criticalIssues, 2)

				_, deletedIssueFound := lo.Find(criticalIssues, func(i billingadapter.ValidationIssueWithDBMeta) bool {
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
				defer s.InvoiceCalculator.DisableMock(t)

				// Given that the app will return a validation error
				mockApp.OnValidateInvoice(billingentity.NewValidationWarning("test1", "validation warning"))
				mockApp.OnUpsertInvoice(nil)
				mockApp.OnFinalizeInvoice(nil)
				calcMock.OnCalculate(nil)

				// When we create a draft invoice
				invoice := s.createDraftInvoice(s.T(), ctx, draftInvoiceInput{
					Namespace: ns,
					Customer:  customer,
				})
				require.NotNil(s.T(), invoice)

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
						Component: "app.sandbox.invoiceCustomers.validate",
					},
				}, invoice.ValidationIssues.RemoveMetaForCompare())

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

				CustomerMutate: customerentity.CustomerMutate{
					Name:         "Test Customer",
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

			mockApp.AssertExpectations(t)

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

func (s *InvoicingTestSuite) TestUBPInvoicing() {
	namespace := "ns-ubp-invoicing"
	ctx := context.Background()

	periodStart := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2024-09-03T12:13:14Z"))

	_ = s.installSandboxApp(s.T(), namespace)

	s.MeterRepo.ReplaceMeters(ctx, []models.Meter{
		{
			Namespace:   namespace,
			Slug:        "flat-per-unit",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
		{
			Namespace:   namespace,
			Slug:        "flat-per-usage",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
		{
			Namespace:   namespace,
			Slug:        "tiered-graduated",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
		{
			Namespace:   namespace,
			Slug:        "tiered-volume",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
	})
	defer s.MeterRepo.ReplaceMeters(ctx, []models.Meter{})

	// Let's initialize the mock streaming connector with data that is out of the period so that we
	// can start with empty values
	for _, slug := range []string{"flat-per-unit", "flat-per-usage", "tiered-graduated", "tiered-volume"} {
		s.MockStreamingConnector.AddSimpleEvent(slug, 0, periodStart.Add(-time.Minute))
	}

	defer s.MockStreamingConnector.Reset()

	// Let's create the features
	// TODO[later]: we need to handle archived features, do we want to issue a warning? Can features be archived when used
	// by a draft invoice?
	features := ubpFeatures{
		flatPerUnit: lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "flat-per-unit",
			Key:       "flat-per-unit",
			MeterSlug: lo.ToPtr("flat-per-unit"),
		})),
		flatPerUsage: lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "flat-per-usage",
			Key:       "flat-per-usage",
			MeterSlug: lo.ToPtr("flat-per-usage"),
		})),
		tieredGraduated: lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "tiered-graduated",
			Key:       "tiered-graduated",
			MeterSlug: lo.ToPtr("tiered-graduated"),
		})),
		tieredVolume: lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "tiered-volume",
			Key:       "tiered-volume",
			MeterSlug: lo.ToPtr("tiered-volume"),
		})),
	}

	// Given we have a test customer

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customerentity.CustomerMutate{
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
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
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

	lines := ubpPendingLines{}
	s.Run("create pending invoice items", func() {
		// When we create pending invoice items
		pendingLines, err := s.BillingService.CreateInvoiceLines(ctx,
			billing.CreateInvoiceLinesInput{
				Namespace:  namespace,
				CustomerID: customerEntity.ID,
				Lines: []billingentity.Line{
					{
						LineBase: billingentity.LineBase{
							Period:    billingentity.Period{Start: periodStart, End: periodEnd},
							InvoiceAt: periodEnd,
							Currency:  currencyx.Code(currency.USD),
							Type:      billingentity.InvoiceLineTypeUsageBased,
							Name:      "UBP - FLAT per unit",
						},
						UsageBased: billingentity.UsageBasedLine{
							FeatureKey: features.flatPerUnit.Key,
							Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
								Amount:        alpacadecimal.NewFromFloat(100),
								MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(2000)),
							}),
						},
					},
					{
						LineBase: billingentity.LineBase{
							Period:    billingentity.Period{Start: periodStart, End: periodEnd},
							InvoiceAt: periodEnd,
							Currency:  currencyx.Code(currency.USD),
							Type:      billingentity.InvoiceLineTypeUsageBased,
							Name:      "UBP - FLAT per any usage",
						},
						UsageBased: billingentity.UsageBasedLine{
							FeatureKey: features.flatPerUsage.Key,
							Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
								Amount:      alpacadecimal.NewFromFloat(100),
								PaymentTerm: productcatalog.InArrearsPaymentTerm,
							}),
						},
					},
					{
						LineBase: billingentity.LineBase{
							Period:    billingentity.Period{Start: periodStart, End: periodEnd},
							InvoiceAt: periodEnd,
							Currency:  currencyx.Code(currency.USD),
							Type:      billingentity.InvoiceLineTypeUsageBased,
							Name:      "UBP - Tiered graduated",
						},
						UsageBased: billingentity.UsageBasedLine{
							FeatureKey: features.tieredGraduated.Key,
							Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
								Mode: productcatalog.GraduatedTieredPrice,
								Tiers: []productcatalog.PriceTier{
									{
										UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(100),
										},
									},
									{
										UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(20)),
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(90),
										},
									},
									{
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(80),
										},
									},
								},
							}),
						},
					},
					{
						LineBase: billingentity.LineBase{
							Period:    billingentity.Period{Start: periodStart, End: periodEnd},
							InvoiceAt: periodEnd,
							Currency:  currencyx.Code(currency.USD),
							Type:      billingentity.InvoiceLineTypeUsageBased,
							Name:      "UBP - Tiered volume",
						},
						UsageBased: billingentity.UsageBasedLine{
							FeatureKey: features.tieredVolume.Key,
							Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
								Mode: productcatalog.VolumeTieredPrice,
								Tiers: []productcatalog.PriceTier{
									{
										UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(100),
										},
									},
									{
										UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(20)),
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(90),
										},
									},
									{
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(80),
										},
									},
								},
								MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(3000)),
							}),
						},
					},
				},
			},
		)
		require.NoError(s.T(), err)
		require.Len(s.T(), pendingLines, 4)

		// The pending invoice items should be truncated to 1 min resolution (start => up to next, end down to previous)
		for _, line := range pendingLines {
			require.Equal(s.T(),
				billingentity.Period{
					Start: lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:00Z")),
					End:   lo.Must(time.Parse(time.RFC3339, "2024-09-03T12:13:00Z")),
				},
				line.Period,
				"period should be truncated to 1 min resolution",
			)

			require.Equal(s.T(),
				line.InvoiceAt,
				periodEnd,
				"invoice at should be unchanged",
			)
		}

		lines = ubpPendingLines{
			flatPerUnit:     pendingLines[0],
			flatPerUsage:    pendingLines[1],
			tieredGraduated: pendingLines[2],
			tieredVolume:    pendingLines[3],
		}
	})

	s.Run("create invoice with empty truncated periods", func() {
		asOf := periodStart.Add(time.Second)
		_, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     &asOf,
		})

		require.ErrorIs(s.T(), err, billingentity.ErrInvoiceCreateNoLines)
		require.ErrorAs(s.T(), err, &billingentity.ValidationError{})
	})

	s.Run("create mid period invoice", func() {
		// Usage
		s.MockStreamingConnector.AddSimpleEvent("flat-per-unit", 10, periodStart)

		// Period
		asOf := periodStart.Add(time.Hour)
		out, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     &asOf,
		})

		require.NoError(s.T(), err)
		require.Len(s.T(), out, 1)

		require.Len(s.T(), out[0].ValidationIssues, 0)

		invoiceLines := out[0].Lines.MustGet()
		require.Len(s.T(), invoiceLines, 3)

		// Let's resolve the lines by parent
		flatPerUnit := s.lineWithParent(invoiceLines, lines.flatPerUnit.ID)
		flatPerUsage := s.lineWithParent(invoiceLines, lines.flatPerUsage.ID)
		tieredGraduated := s.lineWithParent(invoiceLines, lines.tieredGraduated.ID)

		// The invoice should not have:
		// - the volume item as that must be invoiced in arreas
		require.NotContains(s.T(), lo.Map(invoiceLines, func(l *billingentity.Line, _ int) string {
			return l.ID
		}), []string{
			flatPerUnit.ID,
			flatPerUsage.ID,
			tieredGraduated.ID,
		})

		expectedPeriod := billingentity.Period{
			Start: periodStart.Truncate(time.Minute),
			End:   periodStart.Add(time.Hour).Truncate(time.Minute),
		}
		for _, line := range invoiceLines {
			require.True(s.T(), expectedPeriod.Equal(line.Period), "period should be changed for the line items")
		}

		// Let's validate the output of the split itself
		tieredGraduatedChildren := s.getLineChildLines(ctx, namespace, lines.tieredGraduated.ID)
		require.True(s.T(), tieredGraduatedChildren.ParentLine.Period.Equal(lines.tieredGraduated.Period))
		require.Equal(s.T(), flatPerUnit.UsageBased.Quantity.InexactFloat64(), float64(10), "flat per unit should have 10 units")
		require.Equal(s.T(), billingentity.InvoiceLineStatusSplit, tieredGraduatedChildren.ParentLine.Status, "parent should be split [id=%s]", tieredGraduatedChildren.ParentLine.ID)
		require.Len(s.T(), tieredGraduatedChildren.ChildLines, 2, "there should be to child lines [id=%s]", tieredGraduatedChildren.ParentLine.ID)
		require.True(s.T(), tieredGraduatedChildren.ChildLines[0].Period.Equal(billingentity.Period{
			Start: periodStart.Truncate(time.Minute),
			End:   periodStart.Add(time.Hour).Truncate(time.Minute),
		}), "first child period should be truncated")
		require.True(s.T(), tieredGraduatedChildren.ChildLines[0].InvoiceAt.Equal(periodStart.Add(time.Hour).Truncate(time.Minute)), "first child should be issued at the end of parent's period")
		require.True(s.T(), tieredGraduatedChildren.ChildLines[1].Period.Equal(billingentity.Period{
			Start: periodStart.Add(time.Hour).Truncate(time.Minute),
			End:   periodEnd.Truncate(time.Minute),
		}), "second child period should be until the end of parent's period")

		// Let's validate detailed line items
		requireDetailedLines(s.T(), flatPerUnit, lineExpectations{
			Details: map[string]feeLineExpect{
				lineservice.UnitPriceUsageChildUniqueReferenceID: {
					Quantity:      10,
					PerUnitAmount: 100,
				},
			},
		})

		// Let's validate the totals
		requireTotals(s.T(), expectedTotals{
			Amount: 1000,
			Total:  1000,
		}, flatPerUnit.Children.MustGet()[0].Totals)

		requireTotals(s.T(), expectedTotals{
			Amount: 1000,
			Total:  1000,
		}, flatPerUnit.Totals)

		requireTotals(s.T(), expectedTotals{
			Amount: 1000,
			Total:  1000,
		}, out[0].Totals)

		s.Run("update line item", func() {
			line, err := s.BillingService.UpdateInvoiceLine(ctx, billing.UpdateInvoiceLineInput{
				Line: billingentity.LineID{
					Namespace: namespace,
					ID:        flatPerUnit.ID,
				},
				Type: billingentity.InvoiceLineTypeUsageBased,
				UsageBased: billing.UpdateInvoiceLineUsageBasedInput{
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(250),
					}),
				},
			})
			require.NoError(s.T(), err)
			require.NotNil(s.T(), line)

			// TODO[later]: we need to decide how to handle the situation where the line is updated, but there are split
			// lines
			require.Equal(s.T(), float64(250), lo.Must(line.UsageBased.Price.AsUnit()).Amount.InexactFloat64())
			require.True(s.T(), flatPerUnit.UpdatedAt.Before(line.UpdatedAt), "updated at should be changed")
			require.True(s.T(), flatPerUnit.CreatedAt.Equal(line.CreatedAt), "created at should not be changed")

			requireTotals(s.T(), expectedTotals{
				Amount: 2500,
				Total:  2500,
			}, line.Totals)

			invoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: billingentity.InvoiceID{
					Namespace: namespace,
					ID:        out[0].ID,
				},
				Expand: billingentity.InvoiceExpand{},
			})
			require.NoError(s.T(), err)

			requireTotals(s.T(), expectedTotals{
				Amount: 2500,
				Total:  2500,
			}, invoice.Totals)
		})

		s.Run("invalid update of a line item", func() {
			line, err := s.BillingService.UpdateInvoiceLine(ctx, billing.UpdateInvoiceLineInput{
				Line: billingentity.LineID{
					Namespace: namespace,
					ID:        flatPerUnit.ID,
				},
				Type: billingentity.InvoiceLineTypeUsageBased,
				UsageBased: billing.UpdateInvoiceLineUsageBasedInput{
					Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
						Mode: productcatalog.VolumeTieredPrice,
						Tiers: []productcatalog.PriceTier{
							{
								UnitPrice: &productcatalog.PriceTierUnitPrice{
									Amount: alpacadecimal.NewFromFloat(250),
								},
							},
						},
					}),
				},
			})

			require.Error(s.T(), err)
			require.ErrorAs(s.T(), err, &billingentity.ValidationError{})
			require.ErrorIs(s.T(), err, billingentity.ErrInvoiceLinesNotBillable)
			require.Nil(s.T(), line)
		})

		s.Run("deleting a detailed line item would fail", func() {
			detailedLine := flatPerUnit.Children.MustGet()[0]

			err := s.BillingService.DeleteInvoiceLine(ctx, detailedLine.LineID())
			require.Error(s.T(), err)
			require.ErrorIs(s.T(), err, billingentity.ErrInvoiceLineDeleteInvalidStatus)
			require.ErrorAs(s.T(), err, &billingentity.ValidationError{})
		})

		s.Run("deleting a valid line item worked", func() {
			err := s.BillingService.DeleteInvoiceLine(ctx, flatPerUnit.LineID())
			require.NoError(s.T(), err)

			invoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: billingentity.InvoiceID{
					Namespace: namespace,
					ID:        out[0].ID,
				},
				Expand: billingentity.InvoiceExpandAll.SetDeletedLines(true),
			})
			require.NoError(s.T(), err)

			require.Len(s.T(), invoice.Lines.MustGet(), 3)

			deletedLine := invoice.Lines.GetByID(flatPerUnit.ID)
			require.NotNil(s.T(), deletedLine)
			require.NotNil(s.T(), deletedLine.DeletedAt)

			requireTotals(s.T(), expectedTotals{
				Amount: 0,
				Total:  0,
			}, invoice.Totals)

			// Let's validate without deleted line fetching
			invoice, err = s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: out[0].InvoiceID(),
				Expand:  billingentity.InvoiceExpandAll.SetDeletedLines(false),
			})
			require.NoError(s.T(), err)

			require.NotContains(s.T(), lo.Map(invoice.Lines.MustGet(), func(l *billingentity.Line, _ int) string {
				return l.ID
			}), []string{flatPerUnit.ID})

			requireTotals(s.T(), expectedTotals{
				Amount: 0,
				Total:  0,
			}, invoice.Totals)
		})

		s.Run("invoice deletion works", func() {
			// Mock invoicing app
			mockApp := s.SandboxApp.EnableMock(s.T())
			defer s.SandboxApp.DisableMock()

			s.Run("when a validation error occurs, the error is returned", func() {
				// InvoiceDeletion fails
				validationError := billingentity.NewValidationError("delete-failed", "invoice cannot be deleted")
				mockApp.OnDeleteInvoice(validationError)

				err := s.BillingService.DeleteInvoice(ctx, out[0].InvoiceID())
				require.Error(s.T(), err)
				require.ErrorAs(s.T(), err, &billingentity.ValidationError{})

				validationIssue := billingentity.ValidationIssue{}
				require.True(s.T(), errors.As(err, &validationIssue))
				require.Equal(s.T(), validationIssue.Code, validationError.Code)
				require.Equal(s.T(), validationIssue.Message, validationError.Message)

				deletedInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
					Invoice: out[0].InvoiceID(),
					Expand:  billingentity.InvoiceExpandAll,
				})
				require.NoError(s.T(), err)
				require.NotNil(s.T(), deletedInvoice.DeletedAt)
				require.Equal(s.T(), billingentity.InvoiceStatusDeleteFailed, deletedInvoice.Status)

				mockApp.AssertExpectations(s.T())
			})

			s.Run("when a generic error occurs, the error is added to the validation errors", func() {
				mockApp.Reset(s.T())

				// InvoiceDeletion fails
				mockApp.OnDeleteInvoice(errors.New("generic error"))

				invoice, err := s.BillingService.RetryInvoice(ctx, out[0].InvoiceID())
				require.NotNil(s.T(), invoice)
				require.NoError(s.T(), err)
				require.Len(s.T(), invoice.ValidationIssues, 1)
				require.Equal(s.T(), billingentity.InvoiceStatusDeleteFailed, invoice.Status)

				validationIssue := invoice.ValidationIssues[0]
				require.Empty(s.T(), validationIssue.Code)
				require.Equal(s.T(), "generic error", validationIssue.Message)
				require.Equal(s.T(), billingentity.ValidationIssueSeverityCritical, validationIssue.Severity)

				mockApp.AssertExpectations(s.T())
			})

			s.Run("when the sync passes, the invoice is deleted", func() {
				mockApp.Reset(s.T())

				mockApp.OnDeleteInvoice(nil)

				invoice, err := s.BillingService.RetryInvoice(ctx, out[0].InvoiceID())
				require.NotNil(s.T(), invoice)
				require.NoError(s.T(), err)
				require.Len(s.T(), invoice.ValidationIssues, 0)
				require.Equal(s.T(), billingentity.InvoiceStatusDeleted, invoice.Status)

				mockApp.AssertExpectations(s.T())
			})
		})
	})

	s.Run("create mid period invoice - pt2", func() {
		// Mock invoicing app
		mockApp := s.SandboxApp.EnableMock(s.T())
		defer s.SandboxApp.DisableMock()

		mockApp.OnValidateInvoice(nil)
		mockApp.OnUpsertInvoice(func(i billingentity.Invoice) *billingentity.UpsertInvoiceResult {
			lines := i.FlattenLinesByID()

			out := billingentity.NewUpsertInvoiceResult()

			for _, line := range lines {
				if line.Type == billingentity.InvoiceLineTypeFee {
					out.AddLineExternalID(line.ID, line.ID)
				}
			}

			out.SetInvoiceNumber("INV-123")

			return out
		})

		// Usage
		s.MockStreamingConnector.AddSimpleEvent("flat-per-unit", 20, periodStart.Add(time.Minute*100))
		s.MockStreamingConnector.AddSimpleEvent("tiered-graduated", 15, periodStart.Add(time.Minute*100))

		asOf := periodStart.Add(2 * time.Hour)
		out, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     &asOf,
		})

		require.NoError(s.T(), err)
		require.Len(s.T(), out, 1)
		require.Len(s.T(), out[0].ValidationIssues, 0)

		invoiceLines := out[0].Lines.MustGet()

		require.Len(s.T(), invoiceLines, 3)

		// Let's resolve the lines by parent
		flatPerUnit := s.lineWithParent(invoiceLines, lines.flatPerUnit.ID)
		flatPerUsage := s.lineWithParent(invoiceLines, lines.flatPerUsage.ID)
		tieredGraduated := s.lineWithParent(invoiceLines, lines.tieredGraduated.ID)

		// The invoice should not have:
		// - the volume item as that must be invoiced in arreas
		require.NotContains(s.T(), lo.Map(invoiceLines, func(l *billingentity.Line, _ int) string {
			return l.ID
		}), []string{
			flatPerUnit.ID,
			flatPerUsage.ID,
			tieredGraduated.ID,
		})

		expectedPeriod := billingentity.Period{
			Start: periodStart.Add(time.Hour).Truncate(time.Minute),
			End:   periodStart.Add(2 * time.Hour).Truncate(time.Minute),
		}
		for _, line := range invoiceLines {
			require.True(s.T(), expectedPeriod.Equal(line.Period), "period should be changed for the line items")
		}

		// Let's validate the output of the split itself
		tieredGraduatedChildren := s.getLineChildLines(ctx, namespace, lines.tieredGraduated.ID)
		require.True(s.T(), tieredGraduatedChildren.ParentLine.Period.Equal(lines.tieredGraduated.Period))
		require.Equal(s.T(), billingentity.InvoiceLineStatusSplit, tieredGraduatedChildren.ParentLine.Status, "parent should be split [id=%s]", tieredGraduatedChildren.ParentLine.ID)
		require.Len(s.T(), tieredGraduatedChildren.ChildLines, 3, "there should be to child lines [id=%s]", tieredGraduatedChildren.ParentLine.ID)
		require.True(s.T(), tieredGraduatedChildren.ChildLines[0].Period.Equal(billingentity.Period{
			Start: periodStart.Truncate(time.Minute),
			End:   periodStart.Add(time.Hour).Truncate(time.Minute),
		}), "first child period should be truncated")
		require.True(s.T(), tieredGraduatedChildren.ChildLines[1].Period.Equal(billingentity.Period{
			Start: periodStart.Add(time.Hour).Truncate(time.Minute),
			End:   periodStart.Add(2 * time.Hour).Truncate(time.Minute),
		}), "second child period should be between the first and the third child's period")
		require.True(s.T(), tieredGraduatedChildren.ChildLines[1].InvoiceAt.Equal(periodStart.Add(2*time.Hour).Truncate(time.Minute)), "second child should be issued at the end of parent's period")
		require.True(s.T(), tieredGraduatedChildren.ChildLines[2].Period.Equal(billingentity.Period{
			Start: periodStart.Add(2 * time.Hour).Truncate(time.Minute),
			End:   periodEnd.Truncate(time.Minute),
		}), "third child period should be until the end of parent's period")

		// Detailed lines
		requireDetailedLines(s.T(), flatPerUnit, lineExpectations{
			Details: map[string]feeLineExpect{
				lineservice.UnitPriceUsageChildUniqueReferenceID: {
					Quantity:      20,
					PerUnitAmount: 100,
					Discounts: map[string]float64{
						billingentity.LineMaximumSpendReferenceID: 1000,
					},
				},
			},
		})

		requireDetailedLines(s.T(), tieredGraduated, lineExpectations{
			Details: map[string]feeLineExpect{
				fmt.Sprintf(lineservice.GraduatedTieredPriceUsageChildUniqueReferenceID, 1): {
					Quantity:      10,
					PerUnitAmount: 100,
				},
				fmt.Sprintf(lineservice.GraduatedTieredPriceUsageChildUniqueReferenceID, 2): {
					Quantity:      5,
					PerUnitAmount: 90,
				},
			},
		})

		// Let's validate the totals
		requireTotals(s.T(), expectedTotals{
			Amount:         2000,
			DiscountsTotal: 1000,
			Total:          1000,
		}, flatPerUnit.Totals)

		requireTotals(s.T(), expectedTotals{
			Amount: 1450,
			Total:  1450,
		}, tieredGraduated.Totals)

		requireTotals(s.T(), expectedTotals{
			Amount:         3450,
			DiscountsTotal: 1000,
			Total:          2450,
		}, out[0].Totals)

		// Invoice app testing

		require.Equal(s.T(), "INV-123", *out[0].Number)

		for _, line := range out[0].FlattenLinesByID() {
			switch {
			case line.Type == billingentity.InvoiceLineTypeFee:
				require.Equal(s.T(), line.ID, line.ExternalIDs.Invoicing)
			case line.Type == billingentity.InvoiceLineTypeUsageBased:
				require.Empty(s.T(), line.ExternalIDs.Invoicing)
			default:
				s.T().Errorf("unexpected line type: %s", line.Type)
			}
		}

		mockApp.AssertExpectations(s.T())

		s.Run("validate invoice finalization", func() {
			mockApp.OnUpsertInvoice(func(i billingentity.Invoice) *billingentity.UpsertInvoiceResult {
				lines := i.FlattenLinesByID()

				out := billingentity.NewUpsertInvoiceResult()

				for _, line := range lines {
					if line.Type == billingentity.InvoiceLineTypeFee {
						out.AddLineExternalID(line.ID, "final_upsert_"+line.ID)
					}
				}

				return out
			})

			finalizedInvoiceResult := billingentity.NewFinalizeInvoiceResult()
			finalizedInvoiceResult.SetPaymentExternalID("payment_external_id")
			mockApp.OnFinalizeInvoice(finalizedInvoiceResult)

			// Let's finalize the invoice
			finalizedInvoice, err := s.BillingService.ApproveInvoice(ctx, out[0].InvoiceID())
			require.NoError(s.T(), err)
			require.NotNil(s.T(), finalizedInvoice)

			require.Equal(s.T(), "payment_external_id", finalizedInvoice.ExternalIDs.Payment)
			// Invoice app testing
			for _, line := range finalizedInvoice.FlattenLinesByID() {
				switch {
				case line.Type == billingentity.InvoiceLineTypeFee:
					require.Equal(s.T(), "final_upsert_"+line.ID, line.ExternalIDs.Invoicing)
				case line.Type == billingentity.InvoiceLineTypeUsageBased:
					require.Empty(s.T(), line.ExternalIDs.Invoicing)
				default:
					s.T().Errorf("unexpected line type: %s", line.Type)
				}
			}

			mockApp.AssertExpectations(s.T())
		})
	})

	s.Run("create end of period invoice", func() {
		// Usage
		afterPreviousTest := periodStart.Add(3 * time.Hour)
		s.MockStreamingConnector.AddSimpleEvent("tiered-volume", 25, afterPreviousTest)
		s.MockStreamingConnector.AddSimpleEvent("tiered-graduated", 15, afterPreviousTest)

		asOf := periodEnd
		out, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     &asOf,
		})

		require.NoError(s.T(), err)
		require.Len(s.T(), out, 1)

		invoiceLines := out[0].Lines.MustGet()

		require.Len(s.T(), invoiceLines, 4)

		// Let's resolve the lines by parent
		flatPerUnit := s.lineWithParent(invoiceLines, lines.flatPerUnit.ID)
		flatPerUsage := s.lineWithParent(invoiceLines, lines.flatPerUsage.ID)
		tieredGraduated := s.lineWithParent(invoiceLines, lines.tieredGraduated.ID)
		tieredVolume, tieredVolumeFound := lo.Find(invoiceLines, func(l *billingentity.Line) bool {
			return l.ID == lines.tieredVolume.ID
		})
		require.True(s.T(), tieredVolumeFound, "tiered volume line should be present")
		require.Equal(s.T(), tieredVolume.ID, lines.tieredVolume.ID, "tiered volume line should be the same (no split occurred)")

		require.NotContains(s.T(), lo.Map(invoiceLines, func(l *billingentity.Line, _ int) string {
			return l.ID
		}), []string{
			flatPerUnit.ID,
			flatPerUsage.ID,
			tieredGraduated.ID,
			lines.tieredVolume.ID,
		})

		expectedPeriod := billingentity.Period{
			Start: periodStart.Add(2 * time.Hour).Truncate(time.Minute),
			End:   periodEnd.Truncate(time.Minute),
		}
		for _, line := range []*billingentity.Line{flatPerUnit, flatPerUsage, tieredGraduated} {
			require.True(s.T(), expectedPeriod.Equal(line.Period), "period should be changed for the line items")
		}
		require.True(s.T(), tieredVolume.Period.Equal(lines.tieredVolume.Period), "period should be unchanged for the tiered volume line")

		// Let's validate the output of the split itself: no new split should have occurred
		tieredGraduatedChildren := s.getLineChildLines(ctx, namespace, lines.tieredGraduated.ID)
		require.True(s.T(), tieredGraduatedChildren.ParentLine.Period.Equal(lines.tieredGraduated.Period))
		require.Equal(s.T(), billingentity.InvoiceLineStatusSplit, tieredGraduatedChildren.ParentLine.Status, "parent should be split [id=%s]", tieredGraduatedChildren.ParentLine.ID)
		require.Len(s.T(), tieredGraduatedChildren.ChildLines, 3, "there should be to child lines [id=%s]", tieredGraduatedChildren.ParentLine.ID)
		require.True(s.T(), tieredGraduatedChildren.ChildLines[0].Period.Equal(billingentity.Period{
			Start: periodStart.Truncate(time.Minute),
			End:   periodStart.Add(time.Hour).Truncate(time.Minute),
		}), "first child period should be truncated")
		require.True(s.T(), tieredGraduatedChildren.ChildLines[1].Period.Equal(billingentity.Period{
			Start: periodStart.Add(time.Hour).Truncate(time.Minute),
			End:   periodStart.Add(2 * time.Hour).Truncate(time.Minute),
		}), "second child period should be between the first and the third child's period")
		require.True(s.T(), tieredGraduatedChildren.ChildLines[1].InvoiceAt.Equal(periodStart.Add(2*time.Hour).Truncate(time.Minute)), "second child should be issued at the end of parent's period")
		require.True(s.T(), tieredGraduatedChildren.ChildLines[2].Period.Equal(billingentity.Period{
			Start: periodStart.Add(2 * time.Hour).Truncate(time.Minute),
			End:   periodEnd.Truncate(time.Minute),
		}), "third child period should be until the end of parent's period")

		// Details
		requireDetailedLines(s.T(), flatPerUsage, lineExpectations{
			Details: map[string]feeLineExpect{
				lineservice.FlatPriceChildUniqueReferenceID: {
					Quantity:      1,
					PerUnitAmount: 100,
				},
			},
		})

		requireTotals(s.T(), expectedTotals{
			Amount: 100,
			Total:  100,
		}, flatPerUsage.Totals)

		requireDetailedLines(s.T(), tieredVolume, lineExpectations{
			Details: map[string]feeLineExpect{
				lineservice.VolumeUnitPriceChildUniqueReferenceID: {
					Quantity:      25,
					PerUnitAmount: 80,
				},
				lineservice.VolumeMinSpendChildUniqueReferenceID: {
					Quantity:      1,
					PerUnitAmount: 1000,
				},
			},
		})

		requireTotals(s.T(), expectedTotals{
			Amount:       2000,
			ChargesTotal: 1000,
			Total:        3000,
		}, tieredVolume.Totals)

		requireDetailedLines(s.T(), tieredGraduated, lineExpectations{
			Details: map[string]feeLineExpect{
				fmt.Sprintf(lineservice.GraduatedTieredPriceUsageChildUniqueReferenceID, 2): {
					Quantity:      5,
					PerUnitAmount: 90,
				},
				fmt.Sprintf(lineservice.GraduatedTieredPriceUsageChildUniqueReferenceID, 3): {
					Quantity:      10,
					PerUnitAmount: 80,
				},
			},
		})

		requireTotals(s.T(), expectedTotals{
			Amount: 1250,
			Total:  1250,
		}, tieredGraduated.Totals)

		// invoice totals
		requireTotals(s.T(), expectedTotals{
			Amount:       3350,
			ChargesTotal: 1000,
			Total:        4350,
		}, out[0].Totals)
	})
}

func (s *InvoicingTestSuite) lineWithParent(lines []*billingentity.Line, parentID string) *billingentity.Line {
	s.T().Helper()
	for _, line := range lines {
		if line.ParentLineID != nil && *line.ParentLineID == parentID {
			return line
		}
	}

	require.Fail(s.T(), "line with parent not found")
	return nil
}

type getChlildLinesResponse struct {
	ParentLine *billingentity.Line
	ChildLines []*billingentity.Line
}

func (s *InvoicingTestSuite) getLineChildLines(ctx context.Context, ns string, parentID string) getChlildLinesResponse {
	res, err := s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
		Namespace:                  ns,
		ParentLineIDs:              []string{parentID},
		ParentLineIDsIncludeParent: true,
	})
	require.NoError(s.T(), err)

	if len(res) == 0 {
		require.Fail(s.T(), "no child lines found")
	}

	response := getChlildLinesResponse{}

	for _, line := range res {
		if line.ID == parentID {
			response.ParentLine = line
		} else {
			response.ChildLines = append(response.ChildLines, line)
		}
	}

	slices.SortFunc(response.ChildLines, func(a, b *billingentity.Line) int {
		switch {
		case a.Period.Start.Equal(b.Period.Start):
			return 0
		case a.Period.Start.Before(b.Period.Start):
			return -1
		default:
			return 1
		}
	})

	require.NotEmpty(s.T(), response.ParentLine.ID)
	return response
}

type ubpPendingLines struct {
	flatPerUnit     *billingentity.Line
	flatPerUsage    *billingentity.Line
	tieredGraduated *billingentity.Line
	tieredVolume    *billingentity.Line
}

type ubpFeatures struct {
	flatPerUnit     feature.Feature
	flatPerUsage    feature.Feature
	tieredGraduated feature.Feature
	tieredVolume    feature.Feature
}

type lineExpectations struct {
	Details map[string]feeLineExpect
}

type feeLineExpect struct {
	Quantity      float64
	PerUnitAmount float64
	Discounts     map[string]float64
}

func requireDetailedLines(t *testing.T, line *billingentity.Line, expectations lineExpectations) {
	t.Helper()
	require.NotNil(t, line)
	children := line.Children.MustGet()

	require.Len(t, children, len(expectations.Details))

	detailsById := lo.GroupBy(children, func(l *billingentity.Line) string {
		return *l.ChildUniqueReferenceID
	})

	for key, expect := range expectations.Details {
		require.Contains(t, detailsById, key, "detail %s should be present", key)
		detail := detailsById[key][0]

		require.Equal(t, detail.Type, billingentity.InvoiceLineTypeFee, "line type should be fee")
		require.Equal(t, expect.Quantity, detail.FlatFee.Quantity.InexactFloat64(), "quantity should match")
		require.Equal(t, expect.PerUnitAmount, detail.FlatFee.PerUnitAmount.InexactFloat64(), "per unit amount should match")

		discounts := detail.Discounts.MustGet()
		require.Len(t, discounts, len(expect.Discounts), "discounts should match")

		discountsById := lo.GroupBy(discounts, func(d billingentity.LineDiscount) string {
			return *d.ChildUniqueReferenceID
		})

		for discountType, discountExpect := range expect.Discounts {
			require.Contains(t, discountsById, discountType, "discount %s should be present", discountType)
			discount := discountsById[discountType][0]

			require.Equal(t, discountExpect, discount.Amount.InexactFloat64(), "discount amount should match")
		}
	}
}

type expectedTotals struct {
	// Amount is the total amount value of the line before taxes, discounts and commitments
	Amount float64 `json:"amount"`
	// ChargesTotal is the amount of value of the line that are due to additional charges
	ChargesTotal float64 `json:"chargesTotal"`
	// DiscountsTotal is the amount of value of the line that are due to discounts
	DiscountsTotal float64 `json:"discountsTotal"`

	// TaxesInclusiveTotal is the total amount of taxes that are included in the line
	TaxesInclusiveTotal float64 `json:"taxesInclusiveTotal"`
	// TaxesExclusiveTotal is the total amount of taxes that are excluded from the line
	TaxesExclusiveTotal float64 `json:"taxesExclusiveTotal"`
	// TaxesTotal is the total amount of taxes that are included in the line
	TaxesTotal float64 `json:"taxesTotal"`

	// Total is the total amount value of the line after taxes, discounts and commitments
	Total float64 `json:"total"`
}

func requireTotals(t *testing.T, expected expectedTotals, totals billingentity.Totals) {
	t.Helper()
	totalsFloat := expectedTotals{
		Amount:              totals.Amount.InexactFloat64(),
		ChargesTotal:        totals.ChargesTotal.InexactFloat64(),
		DiscountsTotal:      totals.DiscountsTotal.InexactFloat64(),
		TaxesInclusiveTotal: totals.TaxesInclusiveTotal.InexactFloat64(),
		TaxesExclusiveTotal: totals.TaxesExclusiveTotal.InexactFloat64(),
		TaxesTotal:          totals.TaxesTotal.InexactFloat64(),
		Total:               totals.Total.InexactFloat64(),
	}

	require.Equal(t, expected, totalsFloat)
}

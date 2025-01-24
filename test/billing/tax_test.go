package billing

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type InvoicingTaxTestSuite struct {
	BaseSuite
}

func TestInvoicingTax(t *testing.T) {
	suite.Run(t, new(InvoicingTaxTestSuite))
}

func (s *InvoicingTaxTestSuite) TestTaxBehaviorProfileSnapshotting() {
	namespace := "ns-tax-profile"
	ctx := context.Background()

	_ = s.InstallSandboxApp(s.T(), namespace)

	customer := s.CreateTestCustomer(namespace, "test")

	minimalCreateProfileInput := MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = namespace
	minimalCreateProfileInput.WorkflowConfig.Invoicing.TaxBehavior = lo.ToPtr(productcatalog.InclusiveTaxBehavior)

	profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

	s.NoError(err)
	s.NotNil(profile)

	s.Run("Profile tax behavior is inclusive in billing profile", func() {
		draftInvoice := s.generateDraftInvoice(ctx, namespace, customer)
		s.Equal(productcatalog.InclusiveTaxBehavior, *draftInvoice.Workflow.Config.Invoicing.TaxBehavior)
	})

	s.Run("Profile tax behavior is not set in billing profile, set in override", func() {
		profile.WorkflowConfig.Invoicing.TaxBehavior = nil
		profile.AppReferences = nil
		_, err = s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(profile.BaseProfile))
		s.NoError(err)

		// Let's validate db persisting
		profile, err = s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: namespace})
		s.NoError(err)
		s.Nil(profile.WorkflowConfig.Invoicing.TaxBehavior)

		override := billing.CreateCustomerOverrideInput{
			Namespace:  namespace,
			CustomerID: customer.ID,
			Invoicing: billing.InvoicingOverrideConfig{
				TaxBehavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			},
		}

		_, err := s.BillingService.CreateCustomerOverride(ctx, override)
		s.NoError(err)

		mappedBillingProfile, err := s.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
			Namespace:  namespace,
			CustomerID: customer.ID,
		})
		s.NoError(err)
		s.NotNil(mappedBillingProfile.Profile.WorkflowConfig.Invoicing.TaxBehavior)
		s.Equal(productcatalog.ExclusiveTaxBehavior, *mappedBillingProfile.Profile.WorkflowConfig.Invoicing.TaxBehavior)

		draftInvoice := s.generateDraftInvoice(ctx, namespace, customer)
		s.NotNil(draftInvoice.Workflow.Config.Invoicing.TaxBehavior)
		s.Equal(productcatalog.ExclusiveTaxBehavior, *draftInvoice.Workflow.Config.Invoicing.TaxBehavior)
	})

	s.Run("Profile tax behavior is not set, invoice inherits it, but can be updated", func() {
		err := s.BillingService.DeleteCustomerOverride(ctx, billing.DeleteCustomerOverrideInput{
			Namespace:  namespace,
			CustomerID: customer.ID,
		})
		s.NoError(err)

		mappedBillingProfile, err := s.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
			Namespace:  namespace,
			CustomerID: customer.ID,
		})
		s.NoError(err)
		s.Nil(mappedBillingProfile.Profile.WorkflowConfig.Invoicing.TaxBehavior)

		draftInvoice := s.generateDraftInvoice(ctx, namespace, customer)
		s.Nil(draftInvoice.Workflow.Config.Invoicing.TaxBehavior)

		// let's update the invoice
		updatedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: draftInvoice.InvoiceID(),
			EditFn: func(invoice *billing.Invoice) error {
				invoice.Workflow.Config.Invoicing.TaxBehavior = lo.ToPtr(productcatalog.InclusiveTaxBehavior)
				return nil
			},
		})
		s.NoError(err)
		s.NotNil(updatedInvoice.Workflow.Config.Invoicing.TaxBehavior)
		s.Equal(productcatalog.InclusiveTaxBehavior, *updatedInvoice.Workflow.Config.Invoicing.TaxBehavior)
	})
}

func (s *InvoicingTaxTestSuite) TestLineSplittingRetainsTaxConfig() {
	namespace := "ns-tax-ubp-details"
	ctx := context.Background()

	now := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z")).Truncate(time.Microsecond).In(time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	_ = s.InstallSandboxApp(s.T(), namespace)

	customer := s.CreateTestCustomer(namespace, "test")

	minimalCreateProfileInput := MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = namespace
	minimalCreateProfileInput.WorkflowConfig.Invoicing.ProgressiveBilling = true

	profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

	s.NoError(err)
	s.NotNil(profile)

	meterSlug := "flat-per-unit"

	s.MeterRepo.ReplaceMeters(ctx, []models.Meter{
		{
			Namespace:   namespace,
			Slug:        meterSlug,
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
	})
	defer s.MeterRepo.ReplaceMeters(ctx, []models.Meter{})

	flatPerUnitFeature, err := s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterSlug: lo.ToPtr(meterSlug),
	})
	s.NoError(err)

	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, now.Add(-time.Minute))
	defer s.MockStreamingConnector.Reset()

	taxConfig := &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		Stripe: &productcatalog.StripeTaxConfig{
			Code: "txcd_10000000",
		},
	}

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreateInvoiceLinesInput{
			Namespace: namespace,
			Lines: []billing.LineWithCustomer{
				{
					Line: billing.Line{
						LineBase: billing.LineBase{
							Namespace: namespace,
							Period:    billing.Period{Start: now, End: now.Add(time.Hour * 24)},

							InvoiceAt: now.Add(time.Hour * 24),
							ManagedBy: billing.ManuallyManagedLine,

							Type: billing.InvoiceLineTypeUsageBased,

							Name:     "Test item - USD",
							Currency: currencyx.Code(currency.USD),

							TaxConfig: taxConfig,

							Metadata: map[string]string{
								"key": "value",
							},
						},
						UsageBased: &billing.UsageBasedLine{
							FeatureKey: flatPerUnitFeature.Key,
							Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
								Amount:        alpacadecimal.NewFromFloat(100),
								MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(2000)),
							}),
						},
					},
					CustomerID: customer.ID,
				},
			},
		},
	)

	s.NoError(err)
	s.Len(res, 1)

	// Let's create a partial invoice
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 100, now.Add(time.Minute))
	clock.SetTime(now.Add(2 * time.Minute))

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	invoice := invoices[0]
	s.DebugDumpInvoice("invoice", invoice)

	invoiceLines := invoice.Lines.OrEmpty()
	s.Len(invoiceLines, 1)

	ubpSplitLine := invoiceLines[0]
	s.NotNil(ubpSplitLine.ParentLineID, "the line is a split line")
	s.Equal(ubpSplitLine.TaxConfig, taxConfig, "tax config is retained")

	ubpSplitLineChildren := ubpSplitLine.Children.OrEmpty()
	s.Len(ubpSplitLineChildren, 1)

	ubpDetailedLine := ubpSplitLineChildren[0]
	s.Equal(ubpDetailedLine.TaxConfig, taxConfig, "tax config is retained in detailed line")
}

func (s *InvoicingTaxTestSuite) generateDraftInvoice(ctx context.Context, namespace string, customer *customerentity.Customer) billing.Invoice {
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreateInvoiceLinesInput{
			Namespace: namespace,
			Lines: []billing.LineWithCustomer{
				{
					Line: billing.Line{
						LineBase: billing.LineBase{
							Namespace: namespace,
							Period:    billing.Period{Start: now, End: now.Add(time.Hour * 24)},

							InvoiceAt: now,
							ManagedBy: billing.ManuallyManagedLine,

							Type: billing.InvoiceLineTypeFee,

							Name:     "Test item - USD",
							Currency: currencyx.Code(currency.USD),

							Metadata: map[string]string{
								"key": "value",
							},
						},
						FlatFee: &billing.FlatFeeLine{
							PerUnitAmount: alpacadecimal.NewFromFloat(100),
							Quantity:      alpacadecimal.NewFromFloat(1),
							Category:      billing.FlatFeeCategoryRegular,
							PaymentTerm:   productcatalog.InAdvancePaymentTerm,
						},
					},
					CustomerID: customer.ID,
				},
			},
		},
	)

	s.NoError(err)
	s.Len(res, 1)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Len(invoices, 1)

	return invoices[0]
}
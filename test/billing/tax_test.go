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
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
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

func (s *InvoicingTaxTestSuite) TestDefaultTaxConfigProfileSnapshotting() {
	namespace := "ns-tax-profile"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	cust := s.CreateTestCustomer(namespace, "test")

	profile := s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
		profile.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			Stripe: &productcatalog.StripeTaxConfig{
				Code: "txcd_10000000",
			},
		}
	}))

	s.Run("Profile default tax config is inclusive in billing profile", func() {
		draftInvoice := s.generateDraftInvoice(ctx, cust)
		s.NotNil(draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig)
		s.Equal(productcatalog.InclusiveTaxBehavior, *draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig.Behavior)
		s.NotNil(draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig.Stripe)
		s.Equal("txcd_10000000", draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig.Stripe.Code)
	})

	s.Run("Profile default tax config is not set in billing profile, set in override", func() {
		profile.WorkflowConfig.Invoicing.DefaultTaxConfig = nil
		profile.AppReferences = nil
		_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(profile.BaseProfile))
		s.NoError(err)

		// Let's validate db persisting
		profile, err = s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: namespace})
		s.NoError(err)
		s.Nil(profile.WorkflowConfig.Invoicing.DefaultTaxConfig)

		override := billing.UpsertCustomerOverrideInput{
			Namespace:  namespace,
			CustomerID: cust.ID,
			Invoicing: billing.InvoicingOverrideConfig{
				DefaultTaxConfig: &productcatalog.TaxConfig{
					Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					Stripe: &productcatalog.StripeTaxConfig{
						Code: "txcd_20000000",
					},
				},
			},
		}

		_, err = s.BillingService.UpsertCustomerOverride(ctx, override)
		s.NoError(err)

		customerOverride, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: namespace,
				ID:        cust.ID,
			},
		})
		s.NoError(err)
		s.NotNil(customerOverride.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig)
		s.Equal(productcatalog.ExclusiveTaxBehavior, *customerOverride.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig.Behavior)
		s.Equal("txcd_20000000", customerOverride.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig.Stripe.Code)

		draftInvoice := s.generateDraftInvoice(ctx, cust)
		s.NotNil(draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig)
		s.Equal(productcatalog.ExclusiveTaxBehavior, *draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig.Behavior)
		s.Equal("txcd_20000000", draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig.Stripe.Code)
	})

	s.Run("Profile default tax config is not set, invoice inherits it, but can be updated", func() {
		err := s.BillingService.DeleteCustomerOverride(ctx, billing.DeleteCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: namespace,
				ID:        cust.ID,
			},
		})
		s.NoError(err)

		customerOverride, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: namespace,
				ID:        cust.ID,
			},
		})
		s.NoError(err)
		s.Nil(customerOverride.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig)

		draftInvoice := s.generateDraftInvoice(ctx, cust)
		s.Nil(draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig)

		// let's update the invoice
		updatedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: draftInvoice.InvoiceID(),
			EditFn: func(invoice *billing.StandardInvoice) error {
				invoice.Workflow.Config.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
					Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					Stripe: &productcatalog.StripeTaxConfig{
						Code: "txcd_30000000",
					},
				}
				return nil
			},
		})
		s.NoError(err)
		s.NotNil(updatedInvoice.Workflow.Config.Invoicing.DefaultTaxConfig.Behavior)
		s.Equal(productcatalog.InclusiveTaxBehavior, *updatedInvoice.Workflow.Config.Invoicing.DefaultTaxConfig.Behavior)
		s.Equal("txcd_30000000", updatedInvoice.Workflow.Config.Invoicing.DefaultTaxConfig.Stripe.Code)
	})
}

func (s *InvoicingTaxTestSuite) TestLineSplittingRetainsTaxConfig() {
	namespace := "ns-tax-ubp-details"
	ctx := context.Background()

	now := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z")).Truncate(time.Microsecond).In(time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	customer := s.CreateTestCustomer(namespace, "test")

	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithProgressiveBilling())

	meterSlug := "flat-per-unit"

	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Meter 1",
			},
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	})
	s.NoError(err, "meter replacement must not return error")

	defer func() {
		err = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
		s.NoError(err, "meter replacement must not return error")
	}()

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
		billing.CreatePendingInvoiceLinesInput{
			Customer: customer.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []*billing.StandardLine{
				{
					StandardLineBase: billing.StandardLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Test item - USD",
						}),
						Period: billing.Period{Start: now, End: now.Add(time.Hour * 24)},

						InvoiceAt: now.Add(time.Hour * 24),
						ManagedBy: billing.ManuallyManagedLine,

						TaxConfig: taxConfig,

						Metadata: map[string]string{
							"key": "value",
						},
					},
					UsageBased: &billing.UsageBasedLine{
						FeatureKey: flatPerUnitFeature.Key,
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
							Commitments: productcatalog.Commitments{
								MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(2000)),
							},
						}),
					},
				},
			},
		},
	)

	s.NoError(err)
	s.Len(res.Lines, 1)

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
	s.NotNil(ubpSplitLine.SplitLineGroupID, "the line is a split line")
	s.Equal(ubpSplitLine.TaxConfig, taxConfig, "tax config is retained")

	ubpSplitLineDetailedLines := ubpSplitLine.DetailedLines
	s.Len(ubpSplitLineDetailedLines, 1)

	ubpDetailedLine := ubpSplitLineDetailedLines[0]
	s.Equal(ubpDetailedLine.TaxConfig, taxConfig, "tax config is retained in detailed line")
}

func (s *InvoicingTaxTestSuite) generateDraftInvoice(ctx context.Context, customer *customer.Customer) billing.StandardInvoice {
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customer.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []*billing.StandardLine{
				billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
					Period: billing.Period{Start: now, End: now.Add(time.Hour * 24)},

					InvoiceAt: now,
					ManagedBy: billing.ManuallyManagedLine,

					Name: "Test item - USD",

					Metadata: map[string]string{
						"key": "value",
					},
					PerUnitAmount: alpacadecimal.NewFromFloat(100),
					PaymentTerm:   productcatalog.InAdvancePaymentTerm,
				}),
			},
		},
	)

	s.NoError(err)
	s.Len(res.Lines, 1)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Len(invoices, 1)

	return invoices[0]
}

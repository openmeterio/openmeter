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

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billinghttpdriver "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/customer"
	billinginvoicelinedb "github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

	// The tax code is seeded via the adapter to simulate a legacy row predating the deprecation gate.
	profile := s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	profileTaxCode := s.TaxCodeForStripeCode(ctx, namespace, "txcd_10000000")
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: profileTaxCode.ID,
	})

	s.Run("Profile default tax config is inclusive in billing profile", func() {
		draftInvoice := s.generateDraftInvoice(ctx, cust)
		s.Equal(&productcatalog.TaxCodeConfig{
			Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			TaxCodeID: profileTaxCode.ID,
		}, draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig)
	})

	s.Run("Selected profile supplies customer tax settings independently of the default", func() {
		profile.WorkflowConfig.Invoicing.DefaultTaxConfig = nil
		profile.AppReferences = nil
		_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(profile.BaseProfile))
		s.NoError(err)

		// Let's validate db persisting
		profile, err = s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: namespace})
		s.NoError(err)
		s.Nil(profile.WorkflowConfig.Invoicing.DefaultTaxConfig)

		selectedInput := MinimalCreateProfileInputTemplate(sandboxApp.GetID())
		selectedInput.Namespace = namespace
		selectedInput.Default = false
		selectedProfile, err := s.BillingService.CreateProfile(ctx, selectedInput)
		s.Require().NoError(err)
		selectedTaxCode := s.TaxCodeForStripeCode(ctx, namespace, "txcd_20000000")
		s.SeedProfileDefaultTaxConfigViaAdapter(ctx, selectedProfile.ProfileID(), &productcatalog.TaxCodeConfig{
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: selectedTaxCode.ID,
		})
		_, err = s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
			Customer:  cust.GetID(),
			ProfileID: selectedProfile.ID,
		})
		s.NoError(err)

		customerOverride, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
			Customer: customer.CustomerID{
				Namespace: namespace,
				ID:        cust.ID,
			},
		})
		s.NoError(err)

		wantDefault := &productcatalog.TaxCodeConfig{
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: selectedTaxCode.ID,
		}
		s.Equal(wantDefault, customerOverride.EffectiveProfile.WorkflowConfig.Invoicing.DefaultTaxConfig)

		draftInvoice := s.generateDraftInvoice(ctx, cust)
		s.Equal(wantDefault, draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig)
	})

	s.Run("Profile default tax config is not set, invoice inherits it, but can be updated", func() {
		err := s.BillingService.ClearCustomerBillingProfile(ctx, billing.ClearCustomerBillingProfileInput{
			Customer: customer.CustomerID{
				Namespace: namespace,
				ID:        cust.ID,
			},
		})
		s.NoError(err)

		customerOverride, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
			Customer: customer.CustomerID{
				Namespace: namespace,
				ID:        cust.ID,
			},
		})
		s.NoError(err)
		s.Nil(customerOverride.EffectiveProfile.WorkflowConfig.Invoicing.DefaultTaxConfig)

		draftInvoice := s.generateDraftInvoice(ctx, cust)
		s.Nil(draftInvoice.Workflow.Config.Invoicing.DefaultTaxConfig)
		s.TaxCodeEnv.ProvisionProviderDefaultTaxCode(s.T(), namespace)

		// An unknown Tax Code is rejected.
		_, err = s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      draftInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.StandardInvoice) error {
				invoice.Workflow.Config.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{
					TaxCodeID: ulid.Make().String(),
				}

				return nil
			},
		})
		s.ErrorContains(err, "not found")

		// let's update the invoice
		editedTaxCode := s.TaxCodeForStripeCode(ctx, namespace, "txcd_30000000")
		updatedInvoice, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      draftInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.StandardInvoice) error {
				invoice.Workflow.Config.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					TaxCodeID: editedTaxCode.ID,
				}

				return nil
			},
		})
		s.NoError(err)
		s.Equal(&productcatalog.TaxCodeConfig{
			Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			TaxCodeID: editedTaxCode.ID,
		}, updatedInvoice.Workflow.Config.Invoicing.DefaultTaxConfig)
	})
}

func (s *InvoicingTaxTestSuite) TestInvoiceDefaultTaxCodeFrozenOnStripeRemap() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-tax-default-remap")

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	cust := s.CreateTestCustomer(namespace, "test")
	profile := s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	taxCode := s.TaxCodeForStripeCode(ctx, namespace, "txcd_10000000")
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: taxCode.ID,
	})

	wantDefault := &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: taxCode.ID,
	}

	var invoice billing.StandardInvoice

	s.Run("given: an invoice created from a profile whose default references the tax code", func() {
		invoice = s.generateDraftInvoice(ctx, cust)

		s.Equal(wantDefault, invoice.Workflow.Config.Invoicing.DefaultTaxConfig)

		// Lines inheriting the default still get its Stripe code, Tax Code ID and Tax Code.
		lines := invoice.Lines.OrEmpty()
		s.Require().Len(lines, 1)
		s.Require().NotNil(lines[0].TaxConfig)
		s.Equal(&productcatalog.StripeTaxConfig{Code: "txcd_10000000"}, lines[0].TaxConfig.Stripe)
		s.Equal(lo.ToPtr(taxCode.ID), lines[0].TaxConfig.TaxCodeID)
		s.Equal(lo.ToPtr(productcatalog.InclusiveTaxBehavior), lines[0].TaxConfig.Behavior)
		s.Require().NotNil(lines[0].TaxConfig.TaxCode)
		s.Equal(taxCode.ID, lines[0].TaxConfig.TaxCode.ID)
	})

	s.Run("when: the tax code is remapped to another Stripe code", func() {
		_, err := s.TaxCodeEnv.Service.UpdateTaxCode(ctx, taxcode.UpdateTaxCodeInput{
			NamespacedID: taxCode.NamespacedID,
			Name:         taxCode.Name,
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_20000000"},
			},
			// The test drives the update directly, not through the API, so the immutability
			// guard doesn't apply.
			AllowAppMappingOverride: true,
		})
		s.Require().NoError(err)
	})

	s.Run("then: the invoice default still references the same tax code", func() {
		reread, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
		})
		s.Require().NoError(err)
		s.Equal(wantDefault, reread.Workflow.Config.Invoicing.DefaultTaxConfig)

		// v1 shows the tax code's current mapping for the default (accepted, see the ADR).
		taxCodes, err := billinghttpdriver.ResolveWorkflowTaxCodes(ctx, s.TaxCodeResolver, namespace, reread.Workflow.Config)
		s.Require().NoError(err)

		apiInvoice, err := billinghttpdriver.MapStandardInvoiceToAPI(reread, taxCodes)
		s.Require().NoError(err)
		s.Require().NotNil(apiInvoice.Workflow.Workflow.Invoicing)
		s.Equal(&api.TaxConfig{
			Behavior:  lo.ToPtr(api.TaxBehaviorInclusive),
			Stripe:    &api.StripeTaxConfig{Code: "txcd_20000000"},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		}, apiInvoice.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	})
}

func (s *InvoicingTaxTestSuite) TestDeletedDefaultTaxCodeIsCopiedIntoInvoices() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-tax-default-deleted")

	// given: a profile default whose tax code was deleted afterwards
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	cust := s.CreateTestCustomer(namespace, "test")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	profile := s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	deleted := s.TaxCodeForStripeCode(ctx, namespace, "txcd_10000000")
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{TaxCodeID: deleted.ID})
	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: deleted.NamespacedID}))

	// when: an invoice is created from it
	invoice := s.generateDraftInvoice(ctx, cust)

	// then: invoice creation doesn't fail; the invoice keeps the deleted ID as its default, and
	// lines inheriting it get its ID and Stripe code
	s.Equal(deleted.ID, invoice.Workflow.Config.Invoicing.DefaultTaxCodeID())

	lines := invoice.Lines.OrEmpty()
	s.Require().Len(lines, 1)
	s.Require().NotNil(lines[0].TaxConfig)
	s.Equal(&productcatalog.StripeTaxConfig{Code: "txcd_10000000"}, lines[0].TaxConfig.Stripe)
	s.Equal(lo.ToPtr(deleted.ID), lines[0].TaxConfig.TaxCodeID)
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
	meterID := ulid.Make().String()

	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: meterID,
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
		MeterID:   lo.ToPtr(meterID),
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
			Currency: currencyx.FiatCode(currency.USD),
			Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Test item - USD",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},

						InvoiceAt: now.Add(time.Hour * 24),
						ManagedBy: billing.ManuallyManagedLine,

						TaxConfig: taxConfig,

						Metadata: map[string]string{
							"key": "value",
						},
						FeatureKey: flatPerUnitFeature.Key,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
							Commitments: productcatalog.Commitments{
								MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(2000)),
							},
						})),
					},
				},
			}),
		},
	)

	s.NoError(err)
	s.Len(res.Lines, 1)
	s.Require().NotNil(res.Lines[0].TaxConfig)
	s.Require().NotNil(res.Lines[0].TaxConfig.TaxCodeID, "TaxCodeID must be resolved before the gathering line is persisted")

	createdTC, err := s.TaxCodeEnv.Service.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
		Namespace: namespace,
		AppType:   app.AppTypeStripe,
		TaxCode:   taxConfig.Stripe.Code,
	})
	s.Require().NoError(err, "TaxCode entity must exist before invoice advancement")
	s.Equal(createdTC.ID, *res.Lines[0].TaxConfig.TaxCodeID)

	dbLine, err := s.DBClient.BillingInvoiceLine.Query().
		Where(billinginvoicelinedb.ID(res.Lines[0].GetID())).
		Only(ctx)
	s.Require().NoError(err)
	s.Require().NotNil(dbLine.TaxCodeID, "gathering line tax_code_id column must be populated")
	s.Equal(createdTC.ID, *dbLine.TaxCodeID)
	s.Require().NotNil(dbLine.TaxBehavior, "gathering line tax_behavior column must be populated")
	s.Equal(productcatalog.ExclusiveTaxBehavior, *dbLine.TaxBehavior)
	s.Equal(createdTC.ID, lo.FromPtr(dbLine.TaxConfig.TaxCodeID), "gathering line JSON tax_code_id must match the column")

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
	s.Require().NotNil(ubpSplitLine.TaxConfig, "tax config is retained")
	s.Equal(taxConfig.Behavior, ubpSplitLine.TaxConfig.Behavior, "tax config behavior is retained")
	s.Equal(taxConfig.Stripe, ubpSplitLine.TaxConfig.Stripe, "tax config stripe is retained")
	s.Require().NotNil(ubpSplitLine.TaxConfig.TaxCodeID, "TaxCodeID is stamped during advancement")

	s.Equal(createdTC.ID, *ubpSplitLine.TaxConfig.TaxCodeID, "TaxCodeID must match the DB entity")
	s.Require().NotNil(ubpSplitLine.TaxConfig.TaxCode, "TaxCode entity must be stamped on line")
	s.Equal(createdTC.ID, ubpSplitLine.TaxConfig.TaxCode.ID, "stamped TaxCode entity must match the DB entity")

	ubpSplitLineDetailedLines := ubpSplitLine.DetailedLines
	s.Len(ubpSplitLineDetailedLines, 1)

	// Verify the normalized tax_code_id column is written in the DB (not just the JSONB).
	dbLine, err = s.DBClient.BillingInvoiceLine.Query().
		Where(billinginvoicelinedb.ID(ubpSplitLine.GetID())).
		Only(ctx)
	s.Require().NoError(err)
	s.Require().NotNil(dbLine.TaxCodeID, "tax_code_id column must be populated on the invoice line row")
	s.Equal(createdTC.ID, *dbLine.TaxCodeID, "tax_code_id column must match the resolved TaxCode entity")
	s.Require().NotNil(dbLine.TaxBehavior, "tax_behavior column must be populated on the invoice line row")
	s.Equal(productcatalog.ExclusiveTaxBehavior, *dbLine.TaxBehavior, "tax_behavior column must match")
}

func (s *InvoicingTaxTestSuite) generateDraftInvoice(ctx context.Context, customer *customer.Customer) billing.StandardInvoice {
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customer.GetID(),
			Currency: currencyx.FiatCode(currency.USD),
			Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
				billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
					Period: timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},

					InvoiceAt: now,
					ManagedBy: billing.ManuallyManagedLine,

					Name: "Test item - USD",

					Metadata: map[string]string{
						"key": "value",
					},
					PerUnitAmount: alpacadecimal.NewFromFloat(100),
					PaymentTerm:   productcatalog.InAdvancePaymentTerm,
				}),
			}),
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

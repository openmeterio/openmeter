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

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
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
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		Stripe: &productcatalog.StripeTaxConfig{
			Code: "txcd_10000000",
		},
	})

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
		s.ProvisionProviderDefaultTaxCode(ctx, namespace)

		// let's update the invoice
		updatedInvoice, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      draftInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
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

// TestUpdateProfileRejectsSoftDeletedDefaultTaxConfig asserts that UpdateProfile rejects a
// billing profile's defaultTaxConfig.taxCodeId once the referenced tax code has been soft
// deleted. taxcode.Service.GetTaxCode intentionally still returns soft-deleted rows by ID (they
// remain resolvable for continuity reads such as invoice snapshotting), so the guard against
// assigning a dead reference as a billing default lives in productcatalog.ResolveTaxConfig's
// IncludeDeleted=false check. The taxCodeId must be echoed back unchanged from the stored
// profile for the update to reach that check at all: profile.InvoicingConfig.
// WithDeprecatedTaxCodeEnforced runs first and rejects any *new* taxCodeId assignment outright,
// so this test seeds the stored reference (mirroring a legacy pre-deprecation row) before
// deleting the tax code and echoing it back unchanged.
func (s *InvoicingTaxTestSuite) TestUpdateProfileRejectsSoftDeletedDefaultTaxConfig() {
	namespace := "ns-tax-profile-soft-deleted"
	ctx := s.T().Context()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	// given:
	// - a billing profile whose stored defaultTaxConfig already references a tax code, seeded via
	//   the adapter to simulate a legacy row created before taxCodeId echoing was deprecated
	// - the referenced tax code is then soft-deleted
	profile := s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	taxCode, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: namespace,
		Key:       "soft-deleted-profile-default",
		Name:      "Soft Deleted Profile Default",
	})
	s.NoError(err)

	seededProfile := s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(taxCode.ID),
	})

	// DeleteTaxCode checks the deleted code against the namespace's organization-default tax
	// codes, so those must be provisioned first even though this tax code isn't one of them.
	s.ProvisionDefaultTaxCodes(ctx, namespace)
	s.NoError(s.TaxCodeService.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: taxCode.ID},
	}))

	// when: the profile is updated while echoing the same (now soft-deleted) taxCodeId
	// unchanged, so the update passes the deprecated-tax-code gate and reaches tax-code
	// resolution
	updateInput := billing.UpdateProfileInput(seededProfile.BaseProfile)
	updateInput.AppReferences = nil

	_, err = s.BillingService.UpdateProfile(ctx, updateInput)

	// then: resolution rejects the soft-deleted reference as a generic validation error
	s.True(models.IsGenericValidationError(err), "expected a generic validation error, got: %v", err)
}

// TestCreateStandardInvoiceFromGatheringLinesResolvesSoftDeletedDefaultTaxConfig pins the
// billing continuity contract in CreateStandardInvoiceFromGatheringLines: it re-derives the
// customer's already persisted profile default tax config with IncludeDeleted=true, so a tax
// code that is the profile default and has since been soft-deleted must not block invoicing.
// Without IncludeDeleted=true here, the same soft-delete gate proven in
// TestUpdateProfileRejectsSoftDeletedDefaultTaxConfig would also block routine invoicing for
// every customer whose profile still points at that code as its default, turning a tax-code
// cleanup into an invoicing outage.
func (s *InvoicingTaxTestSuite) TestCreateStandardInvoiceFromGatheringLinesResolvesSoftDeletedDefaultTaxConfig() {
	namespace := "ns-tax-profile-deleted-default"
	ctx := s.T().Context()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	cust := s.CreateTestCustomer(namespace, "test")

	// given:
	// - organization default tax codes are provisioned so DeleteTaxCode's org-default lookup
	//   below has a row to compare against for this namespace
	// - a billing profile whose default tax config references a tax code that is NOT an
	//   organization default (org-default tax codes reject deletion, see taxcode.Service.DeleteTaxCode)
	s.ProvisionDefaultTaxCodes(ctx, namespace)
	profile := s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	taxConfig := &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		Stripe: &productcatalog.StripeTaxConfig{
			Code: "txcd_10000001",
		},
	}

	seeded := s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), taxConfig)
	seededDefault := seeded.WorkflowConfig.Invoicing.DefaultTaxConfig
	s.Require().NotNil(seededDefault, "seeding must persist the default tax config")
	s.Require().NotNil(seededDefault.TaxCodeID, "seeding must resolve and stamp the tax code id")

	deletedTaxCodeID := *seededDefault.TaxCodeID

	// when:
	// - pending lines are created while the default tax code is still live
	// - the default tax code is then soft-deleted before the standard invoice is generated
	//   from those gathering lines
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: cust.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []billing.GatheringLine{
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
			},
		},
	)
	s.NoError(err)
	s.Len(res.Lines, 1)

	s.Require().NoError(s.TaxCodeService.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: deletedTaxCodeID},
	}), "deleting a non-org-default tax code must succeed")

	// then:
	// - invoice creation from the gathering lines still succeeds
	// - the snapshotted default tax config resolves the deleted code's id and Stripe mapping
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err, "invoice creation must succeed even though the profile default tax code was soft-deleted")
	s.Require().Len(invoices, 1)

	invoice := invoices[0]
	s.Require().NotNil(invoice.Workflow.Config.Invoicing.DefaultTaxConfig)
	s.Require().NotNil(invoice.Workflow.Config.Invoicing.DefaultTaxConfig.TaxCodeID)
	s.Equal(deletedTaxCodeID, *invoice.Workflow.Config.Invoicing.DefaultTaxConfig.TaxCodeID)
	s.Require().NotNil(invoice.Workflow.Config.Invoicing.DefaultTaxConfig.Stripe)
	s.Equal("txcd_10000001", invoice.Workflow.Config.Invoicing.DefaultTaxConfig.Stripe.Code)
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
			Currency: currencyx.Code(currency.USD),
			Lines: []billing.GatheringLine{
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
	s.Require().NotNil(ubpSplitLine.TaxConfig, "tax config is retained")
	s.Equal(taxConfig.Behavior, ubpSplitLine.TaxConfig.Behavior, "tax config behavior is retained")
	s.Equal(taxConfig.Stripe, ubpSplitLine.TaxConfig.Stripe, "tax config stripe is retained")
	s.Require().NotNil(ubpSplitLine.TaxConfig.TaxCodeID, "TaxCodeID is stamped during advancement")

	createdTC, err := s.TaxCodeService.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
		Namespace: namespace,
		AppType:   app.AppTypeStripe,
		TaxCode:   taxConfig.Stripe.Code,
	})
	s.Require().NoError(err, "TaxCode entity must exist in DB")
	s.Equal(createdTC.ID, *ubpSplitLine.TaxConfig.TaxCodeID, "TaxCodeID must match the DB entity")
	s.Require().NotNil(ubpSplitLine.TaxConfig.TaxCode, "TaxCode entity must be stamped on line")
	s.Equal(createdTC.ID, ubpSplitLine.TaxConfig.TaxCode.ID, "stamped TaxCode entity must match the DB entity")

	ubpSplitLineDetailedLines := ubpSplitLine.DetailedLines
	s.Len(ubpSplitLineDetailedLines, 1)

	// Verify the normalized tax_code_id column is written in the DB (not just the JSONB).
	dbLine, err := s.DBClient.BillingInvoiceLine.Query().
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
			Currency: currencyx.Code(currency.USD),
			Lines: []billing.GatheringLine{
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

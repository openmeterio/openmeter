package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billinginvoicedb "github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type TaxCodeDualWriteTestSuite struct {
	BaseSuite
}

func TestTaxCodeDualWrite(t *testing.T) {
	suite.Run(t, new(TaxCodeDualWriteTestSuite))
}

// assertInvoiceLineTaxCode verifies that an invoice line has a fully-stamped TaxCode: both
// the FK (TaxCodeID) and the entity snapshot (TaxCode) are present with the expected code.
func (s *TaxCodeDualWriteTestSuite) assertInvoiceLineTaxCode(line *billing.StandardLine, wantCode string) {
	s.T().Helper()
	s.Require().NotNil(line.TaxConfig)
	s.Require().NotNil(line.TaxConfig.TaxCodeID, "TaxCodeID must be stamped on line")
	s.Require().NotNil(line.TaxConfig.TaxCode, "TaxCode entity must be stamped on line")
	s.Require().NotNil(line.TaxConfig.Stripe)
	s.Equal(wantCode, line.TaxConfig.Stripe.Code)
	mapping, ok := line.TaxConfig.TaxCode.GetAppMapping(app.AppTypeStripe)
	s.True(ok, "TaxCode must have a Stripe app mapping")
	s.Equal(wantCode, mapping.TaxCode)
}

// ── Group A: Profile tax code ID gate and storage ──────────────────────────

// A2: Creating a profile with behavior-only stores no Tax Code.
func (s *TaxCodeDualWriteTestSuite) TestProfileCreateBehaviorOnlyNoFK() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{
			Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		}
	}))

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)

	s.Equal(&productcatalog.TaxCodeConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}, readBack.WorkflowConfig.Invoicing.DefaultTaxConfig)
}

// A3: Creating a profile with no DefaultTaxConfig stores nothing.
func (s *TaxCodeDualWriteTestSuite) TestProfileCreateNilTaxConfigNoFK() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)
	s.Nil(readBack.WorkflowConfig.Invoicing.DefaultTaxConfig)
}

// A3b: Creating a profile with a tax code is rejected by the deprecation gate; nothing is persisted.
func (s *TaxCodeDualWriteTestSuite) TestProfileCreateWithTaxCodeRejected() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	tc := s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000")

	input := MinimalCreateProfileInputTemplate(sandboxApp.GetID())
	input.Namespace = ns
	input.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{
		TaxCodeID: tc.ID,
	}

	_, err := s.BillingService.CreateProfile(ctx, input)
	s.Require().Error(err)

	var validationErr billing.ValidationError
	s.Require().True(errors.As(err, &validationErr), "error must be a billing.ValidationError, got: %v", err)
	s.ErrorContains(err, "deprecated")

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)
	s.Nil(readBack, "no default profile should be persisted after a rejected create")
}

// A3c: Adding a tax code to a profile that has none on update is rejected by the deprecation gate.
func (s *TaxCodeDualWriteTestSuite) TestProfileUpdateAddTaxCodeRejected() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	tc := s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000")

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	profile.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{
		TaxCodeID: tc.ID,
	}
	profile.AppReferences = nil
	_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(profile.BaseProfile))
	s.Require().Error(err)

	var validationErr billing.ValidationError
	s.Require().True(errors.As(err, &validationErr), "error must be a billing.ValidationError, got: %v", err)

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)
	s.Nil(readBack.WorkflowConfig.Invoicing.DefaultTaxConfig, "rejected update must not persist a tax code")
}

// A3d: Only the stored Tax Code ID can be echoed back; a different one is rejected even if it exists.
func (s *TaxCodeDualWriteTestSuite) TestProfileUpdateChangeTaxCodeRejected() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	stored := s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000")
	other := s.TaxCodeForStripeCode(ctx, ns, "txcd_20000000")

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	seeded := s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		TaxCodeID: stored.ID,
	})

	update := seeded.BaseProfile
	update.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{TaxCodeID: other.ID}
	update.AppReferences = nil
	_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(update))

	var validationErr billing.ValidationError
	s.Require().ErrorAs(err, &validationErr)

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)
	s.Equal(stored.ID, readBack.WorkflowConfig.Invoicing.DefaultTaxCodeID())
}

// A3e: Echoing the stored Tax Code ID passes the gate without re-checking the Tax Code, so a
// profile whose default Tax Code was deleted since can still be edited.
func (s *TaxCodeDualWriteTestSuite) TestProfileUpdateEchoesDeletedTaxCode() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	// Deleting a tax code checks it isn't an organization default, which needs the defaults set.
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	stored := s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000")

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	seeded := s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		TaxCodeID: stored.ID,
	})

	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{
		NamespacedID: stored.NamespacedID,
	}))

	update := seeded.BaseProfile
	update.Name = "renamed"
	update.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: stored.ID,
	}
	update.AppReferences = nil
	_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(update))
	s.Require().NoError(err)

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)
	s.Equal("renamed", readBack.Name)
	s.Equal(&productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: stored.ID,
	}, readBack.WorkflowConfig.Invoicing.DefaultTaxConfig)
}

// A4: Updating a profile to behavior-only clears the Tax Code.
func (s *TaxCodeDualWriteTestSuite) TestProfileUpdateClearsTaxCodeFK() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	tc := s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000")

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	seeded := s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		TaxCodeID: tc.ID,
	})
	s.Equal(tc.ID, seeded.WorkflowConfig.Invoicing.DefaultTaxCodeID())

	update := seeded.BaseProfile
	update.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}
	update.AppReferences = nil
	_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(update))
	s.NoError(err)

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)
	s.Equal(&productcatalog.TaxCodeConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}, readBack.WorkflowConfig.Invoicing.DefaultTaxConfig)
}

// A5: Updating a profile to nil DefaultTaxConfig clears both the FK and the behavior column.
func (s *TaxCodeDualWriteTestSuite) TestProfileUpdateToNilClearsBothColumns() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	tc := s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000")

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	seeded := s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: tc.ID,
	})

	update := seeded.BaseProfile
	update.WorkflowConfig.Invoicing.DefaultTaxConfig = nil
	update.AppReferences = nil
	_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(update))
	s.NoError(err)

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)
	s.Nil(readBack.WorkflowConfig.Invoicing.DefaultTaxConfig)

	dbWC, err := s.DBClient.BillingWorkflowConfig.Get(ctx, seeded.WorkflowConfigID)
	s.NoError(err)
	s.Nil(dbWC.TaxCodeID)
	s.Nil(dbWC.TaxBehavior)
	s.Empty(dbWC.InvoiceDefaultTaxSettings)
}

// A6: The adapter reads only the tax_code_id / tax_behavior columns and writes a minimal JSON
// copy (tax_code_id + behavior, no Stripe code), so older binaries reading the JSON see the same
// default and the CHECK constraints hold.
func (s *TaxCodeDualWriteTestSuite) TestWorkflowConfigWritesMinimalJSONCopy() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	tc := s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000")

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	seeded := s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: tc.ID,
	})

	dbWC, err := s.DBClient.BillingWorkflowConfig.Get(ctx, seeded.WorkflowConfigID)
	s.Require().NoError(err)
	s.Equal(productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(tc.ID),
	}, dbWC.InvoiceDefaultTaxSettings)
	s.Equal(lo.ToPtr(tc.ID), dbWC.TaxCodeID)
	s.Equal(lo.ToPtr(productcatalog.InclusiveTaxBehavior), dbWC.TaxBehavior)

	// Columns are the source of truth on read: a legacy Stripe code left in the JSON is ignored.
	_, err = s.DBClient.BillingWorkflowConfig.UpdateOneID(seeded.WorkflowConfigID).
		SetInvoiceDefaultTaxSettings(productcatalog.TaxConfig{
			Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			TaxCodeID: lo.ToPtr(tc.ID),
			Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_99999999"},
		}).
		Save(ctx)
	s.Require().NoError(err)

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.Require().NoError(err)
	s.Equal(&productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: tc.ID,
	}, readBack.WorkflowConfig.Invoicing.DefaultTaxConfig)

	// The invoice's cloned workflow config gets the same minimal copy.
	cust := s.CreateTestCustomer(ns, "test")
	now := time.Now().Truncate(time.Microsecond).UTC()

	_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Namespace:     ns,
				Period:        timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},
				InvoiceAt:     now,
				ManagedBy:     billing.ManuallyManagedLine,
				Name:          "line",
				PerUnitAmount: alpacadecimal.NewFromFloat(100),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		}),
	})
	s.Require().NoError(err)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)

	dbInvoice, err := s.DBClient.BillingInvoice.Query().
		Where(billinginvoicedb.ID(invoices[0].ID)).
		WithBillingWorkflowConfig().
		Only(ctx)
	s.Require().NoError(err)
	s.Equal(productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(tc.ID),
	}, dbInvoice.Edges.BillingWorkflowConfig.InvoiceDefaultTaxSettings)
}

// ── Group D: Invoice snapshotting (end-to-end) ────────────────────────────

// D1: Profile DefaultTaxConfig is merged into a nil-TaxConfig line and entity is stamped.
func (s *TaxCodeDualWriteTestSuite) TestSnapshotTaxCodeIntoLinesOnAdvance() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000").ID,
	})
	cust := s.CreateTestCustomer(ns, "test")

	now := time.Now().Truncate(time.Microsecond).UTC()

	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Namespace:     ns,
				Period:        timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},
				InvoiceAt:     now,
				ManagedBy:     billing.ManuallyManagedLine,
				Name:          "nil-taxconfig line",
				PerUnitAmount: alpacadecimal.NewFromFloat(100),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		}),
	})
	s.NoError(err)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Require().Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)

	s.assertInvoiceLineTaxCode(lines[0], "txcd_10000000")
	s.Require().NotNil(lines[0].TaxConfig.Behavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *lines[0].TaxConfig.Behavior)
}

// D2: A line's own Stripe code takes precedence over the profile DefaultTaxConfig.
func (s *TaxCodeDualWriteTestSuite) TestSnapshotLineOwnCodeTakesPrecedence() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		TaxCodeID: s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000").ID,
	})
	cust := s.CreateTestCustomer(ns, "test")

	now := time.Now().Truncate(time.Microsecond).UTC()

	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: ns,
						Name:      "line with own tax code",
					}),
					ServicePeriod: timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},
					InvoiceAt:     now,
					ManagedBy:     billing.ManuallyManagedLine,
					TaxConfig: &productcatalog.TaxConfig{
						Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
					},
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					})),
				},
			},
		}),
	})
	s.NoError(err)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Require().Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)

	s.assertInvoiceLineTaxCode(lines[0], "txcd_20000000")
}

// D3: A line that already has TaxCodeID pre-stamped keeps it (subscription sync scenario).
func (s *TaxCodeDualWriteTestSuite) TestSnapshotPreservesExistingTaxCodeID() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Seed a profile with the Stripe code via the adapter to materialize the TaxCode entity.
	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	seeded := s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		TaxCodeID: s.TaxCodeForStripeCode(ctx, ns, "txcd_10000000").ID,
	})

	existingID := lo.EmptyableToPtr(seeded.WorkflowConfig.Invoicing.DefaultTaxCodeID())
	s.Require().NotNil(existingID)

	cust := s.CreateTestCustomer(ns, "test")
	now := time.Now().Truncate(time.Microsecond).UTC()

	// Create a pending line with the TaxCodeID already stamped (as subscription sync would do).
	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: ns,
						Name:      "pre-stamped line",
					}),
					ServicePeriod: timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},
					InvoiceAt:     now,
					ManagedBy:     billing.ManuallyManagedLine,
					TaxConfig: &productcatalog.TaxConfig{
						Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
						TaxCodeID: existingID,
					},
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					})),
				},
			},
		}),
	})
	s.NoError(err)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Require().Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)

	s.Require().NotNil(lines[0].TaxConfig)
	s.Require().NotNil(lines[0].TaxConfig.TaxCodeID)
	s.Equal(*existingID, *lines[0].TaxConfig.TaxCodeID, "pre-stamped TaxCodeID must not be overwritten")
	s.NotNil(lines[0].TaxConfig.TaxCode, "entity snapshot should still be stamped from the deps map")
}

// D4: SimulateInvoice (readOnly=true) does not create TaxCode entities for unknown codes.
func (s *TaxCodeDualWriteTestSuite) TestSimulateInvoiceReadOnly() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test")

	now := time.Now().Truncate(time.Microsecond).UTC()

	line := billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},
		InvoiceAt:     now,
		Name:          "simulate line",
		PerUnitAmount: alpacadecimal.NewFromFloat(100),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
		ManagedBy:     billing.ManuallyManagedLine,
	})
	line.TaxConfig = billing.FromProductCatalog(&productcatalog.TaxConfig{
		Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_99999999"},
	})

	result, err := s.BillingService.SimulateInvoice(ctx, billing.SimulateInvoiceInput{
		Namespace:  ns,
		CustomerID: &cust.ID,
		Currency:   currencyx.FiatCode(currency.USD),
		Lines:      billing.NewStandardInvoiceLines([]*billing.StandardLine{line}),
	})
	s.NoError(err)

	lines := result.Lines.OrEmpty()
	s.Require().Len(lines, 1)
	s.Require().NotNil(lines[0].TaxConfig)
	s.Nil(lines[0].TaxConfig.TaxCodeID, "readOnly path must not stamp TaxCodeID for unknown codes")

	// Verify that no TaxCode entity was persisted.
	count, err := s.DBClient.TaxCode.Query().
		Where(taxcodedb.Namespace(ns)).
		Count(ctx)
	s.NoError(err)
	s.Equal(0, count, "SimulateInvoice must not write TaxCode entities to the DB")
}

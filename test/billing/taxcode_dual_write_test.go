package billing

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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

// assertTaxConfigHasStripeCode verifies that a TaxConfig has a non-nil TaxCodeID and the
// correct Stripe code. Used for profile/override assertions where only the FK + JSONB
// fields (not the entity snapshot) are populated.
func (s *TaxCodeDualWriteTestSuite) assertTaxConfigHasStripeCode(cfg *productcatalog.TaxConfig, wantCode string) {
	s.T().Helper()
	s.Require().NotNil(cfg)
	s.Require().NotNil(cfg.TaxCodeID, "TaxCodeID must be set via BackfillTaxConfig")
	s.Require().NotNil(cfg.Stripe, "Stripe must be populated via BackfillTaxConfig")
	s.Equal(wantCode, cfg.Stripe.Code)
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

// ── Group A: Profile dual-write / dual-read ─────────────────────────────────

// A1: Creating a profile with a Stripe code creates a TaxCode entity and stamps TaxCodeID.
func (s *TaxCodeDualWriteTestSuite) TestProfileCreateWritesTaxCodeFK() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}))

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)

	cfg := readBack.WorkflowConfig.Invoicing.DefaultTaxConfig
	s.assertTaxConfigHasStripeCode(cfg, "txcd_10000000")
	s.Require().NotNil(cfg.Behavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *cfg.Behavior)

	// Profile returned from CreateProfile should also already have the FK stamped.
	s.assertTaxConfigHasStripeCode(profile.WorkflowConfig.Invoicing.DefaultTaxConfig, "txcd_10000000")
}

// A2: Creating a profile with behavior-only (no Stripe code) does NOT create a TaxCode entity.
func (s *TaxCodeDualWriteTestSuite) TestProfileCreateBehaviorOnlyNoFK() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		}
	}))

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)

	cfg := readBack.WorkflowConfig.Invoicing.DefaultTaxConfig
	s.Require().NotNil(cfg)
	s.Require().NotNil(cfg.Behavior)
	s.Equal(productcatalog.ExclusiveTaxBehavior, *cfg.Behavior)
	s.Nil(cfg.TaxCodeID, "no TaxCode entity should be created for behavior-only config")
	s.Nil(cfg.Stripe, "Stripe must remain nil when no code was given")
}

// A3: Creating a profile with no DefaultTaxConfig stores nothing.
func (s *TaxCodeDualWriteTestSuite) TestProfileCreateNilTaxConfigNoFK() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)
	s.Nil(readBack.WorkflowConfig.Invoicing.DefaultTaxConfig)
}

// A4: Updating a profile to remove the Stripe code clears the TaxCode FK (stale-FK regression).
func (s *TaxCodeDualWriteTestSuite) TestProfileUpdateClearsTaxCodeFK() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}))

	// Confirm TaxCodeID is set after creation.
	s.assertTaxConfigHasStripeCode(profile.WorkflowConfig.Invoicing.DefaultTaxConfig, "txcd_10000000")

	// Update: switch to behavior-only (no Stripe code) — must clear the FK.
	profile.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}
	profile.AppReferences = nil
	_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(profile.BaseProfile))
	s.NoError(err)

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)

	cfg := readBack.WorkflowConfig.Invoicing.DefaultTaxConfig
	s.Require().NotNil(cfg)
	s.Require().NotNil(cfg.Behavior)
	s.Equal(productcatalog.ExclusiveTaxBehavior, *cfg.Behavior)
	s.Nil(cfg.Stripe, "BackfillTaxConfig must not resurrect Stripe from a cleared FK")
	s.Nil(cfg.TaxCodeID, "FK must be cleared by SetOrClearTaxCodeID")
}

// A5: Updating a profile to nil DefaultTaxConfig clears both the FK and the behavior column.
func (s *TaxCodeDualWriteTestSuite) TestProfileUpdateToNilClearsBothColumns() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}))

	profile.WorkflowConfig.Invoicing.DefaultTaxConfig = nil
	profile.AppReferences = nil
	_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(profile.BaseProfile))
	s.NoError(err)

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)
	s.Nil(readBack.WorkflowConfig.Invoicing.DefaultTaxConfig)
}

// A6: Using the same Stripe code on two profiles in the same namespace reuses the same TaxCode entity.
func (s *TaxCodeDualWriteTestSuite) TestProfileTaxCodeIdempotent() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	taxCfg := func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}

	profileA := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(taxCfg))

	// Create a second non-default profile with the same Stripe code.
	inputB := minimalCreateProfileInputTemplate(sandboxApp.GetID())
	inputB.Namespace = ns
	inputB.Default = false
	inputB.Name = "Profile B"
	inputB.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
		Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
	}
	profileB, err := s.BillingService.CreateProfile(ctx, inputB)
	s.NoError(err)

	idA := profileA.WorkflowConfig.Invoicing.DefaultTaxConfig.TaxCodeID
	idB := profileB.WorkflowConfig.Invoicing.DefaultTaxConfig.TaxCodeID
	s.Require().NotNil(idA)
	s.Require().NotNil(idB)
	s.Equal(*idA, *idB, "GetOrCreateByAppMapping must return the same TaxCode entity for the same Stripe code")
}

// ── Group B: Customer override dual-write / dual-read ──────────────────────

// B1: Upserting a customer override with a Stripe code creates a TaxCode entity and stamps TaxCodeID.
func (s *TaxCodeDualWriteTestSuite) TestOverrideUpsertWritesTaxCodeFK() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test")

	_, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		Invoicing: billing.InvoicingOverrideConfig{
			DefaultTaxConfig: &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
			},
		},
	})
	s.NoError(err)

	override, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{Namespace: ns, ID: cust.ID},
	})
	s.NoError(err)

	cfg := override.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig
	s.assertTaxConfigHasStripeCode(cfg, "txcd_20000000")
	s.Require().NotNil(cfg.Behavior)
	s.Equal(productcatalog.ExclusiveTaxBehavior, *cfg.Behavior)
}

// B2: Updating an override to remove the Stripe code clears the FK (stale-FK regression).
func (s *TaxCodeDualWriteTestSuite) TestOverrideUpdateClearsTaxCodeFK() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test")

	// First upsert: with a Stripe code.
	firstOverride, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		Invoicing: billing.InvoicingOverrideConfig{
			DefaultTaxConfig: &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
			},
		},
	})
	s.NoError(err)
	s.assertTaxConfigHasStripeCode(firstOverride.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig, "txcd_20000000")

	// Second upsert: behavior-only, no Stripe code — must clear the FK.
	_, err = s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		Invoicing: billing.InvoicingOverrideConfig{
			DefaultTaxConfig: &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			},
		},
	})
	s.NoError(err)

	override, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{Namespace: ns, ID: cust.ID},
	})
	s.NoError(err)

	cfg := override.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig
	s.Require().NotNil(cfg)
	s.Nil(cfg.Stripe, "Stripe must be nil after clearing the Stripe code")
	s.Nil(cfg.TaxCodeID, "FK must be cleared")
	s.Require().NotNil(cfg.Behavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *cfg.Behavior)
}

// B3: Deleting an override leaves no stale FK visible through BackfillTaxConfig.
func (s *TaxCodeDualWriteTestSuite) TestOverrideDeleteLeavesNoStaleFK() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test")

	_, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		Invoicing: billing.InvoicingOverrideConfig{
			DefaultTaxConfig: &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
			},
		},
	})
	s.NoError(err)

	err = s.BillingService.DeleteCustomerOverride(ctx, billing.DeleteCustomerOverrideInput{
		Customer: customer.CustomerID{Namespace: ns, ID: cust.ID},
	})
	s.NoError(err)

	override, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{Namespace: ns, ID: cust.ID},
	})
	s.NoError(err)

	// No active override — falls back to the default profile (which has no DefaultTaxConfig).
	s.Nil(override.CustomerOverride, "no active override should remain after deletion")
	s.Nil(override.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig)
}

// ── Group C: Edge case guards ─────────────────────────────────────────────

// C1: An empty Stripe code is a no-op — no TaxCode entity is created, TaxCodeID stays nil.
func (s *TaxCodeDualWriteTestSuite) TestEmptyStripeCodeIsNoOp() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Empty Stripe.Code must pass validation (Validate only rejects non-empty invalid codes).
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			Stripe:   &productcatalog.StripeTaxConfig{Code: ""},
		}
	}))

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)

	cfg := readBack.WorkflowConfig.Invoicing.DefaultTaxConfig
	s.Require().NotNil(cfg)
	s.Nil(cfg.TaxCodeID, "empty Stripe code must not create a TaxCode entity")

	// No TaxCode entity must have been persisted for this namespace.
	count, err := s.DBClient.TaxCode.Query().
		Where(taxcodedb.Namespace(ns)).
		Count(ctx)
	s.NoError(err)
	s.Equal(0, count)
}

// C2: Updating a profile with the same Stripe code is idempotent — no duplicate TaxCode entity.
func (s *TaxCodeDualWriteTestSuite) TestResolveDefaultTaxCodeIdempotentOnUpdate() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}))

	firstID := profile.WorkflowConfig.Invoicing.DefaultTaxConfig.TaxCodeID
	s.Require().NotNil(firstID)

	// Update with the same Stripe code.
	profile.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
		Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
	}
	profile.AppReferences = nil
	_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(profile.BaseProfile))
	s.NoError(err)

	readBack, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
	s.NoError(err)

	secondID := readBack.WorkflowConfig.Invoicing.DefaultTaxConfig.TaxCodeID
	s.Require().NotNil(secondID)
	s.Equal(*firstID, *secondID, "GetOrCreateByAppMapping must be idempotent across updates")

	// Exactly one TaxCode entity should exist in the namespace.
	count, err := s.DBClient.TaxCode.Query().
		Where(taxcodedb.Namespace(ns)).
		Count(ctx)
	s.NoError(err)
	s.Equal(1, count)
}

// ── Group D: Invoice snapshotting (end-to-end) ────────────────────────────

// D1: Profile DefaultTaxConfig is merged into a nil-TaxConfig line and entity is stamped.
func (s *TaxCodeDualWriteTestSuite) TestSnapshotTaxCodeIntoLinesOnAdvance() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}))
	cust := s.CreateTestCustomer(ns, "test")

	now := time.Now().Truncate(time.Microsecond).UTC()

	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []billing.GatheringLine{
			billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Namespace:     ns,
				Period:        billing.Period{Start: now, End: now.Add(time.Hour * 24)},
				InvoiceAt:     now,
				ManagedBy:     billing.ManuallyManagedLine,
				Name:          "nil-taxconfig line",
				PerUnitAmount: alpacadecimal.NewFromFloat(100),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		},
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

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}))
	cust := s.CreateTestCustomer(ns, "test")

	now := time.Now().Truncate(time.Microsecond).UTC()

	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []billing.GatheringLine{
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
		},
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

	// Create a profile with the Stripe code to materialize the TaxCode entity.
	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}))

	existingID := profile.WorkflowConfig.Invoicing.DefaultTaxConfig.TaxCodeID
	s.Require().NotNil(existingID)

	cust := s.CreateTestCustomer(ns, "test")
	now := time.Now().Truncate(time.Microsecond).UTC()

	// Create a pending line with the TaxCodeID already stamped (as subscription sync would do).
	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []billing.GatheringLine{
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
		},
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
		Period:        billing.Period{Start: now, End: now.Add(time.Hour * 24)},
		InvoiceAt:     now,
		Name:          "simulate line",
		PerUnitAmount: alpacadecimal.NewFromFloat(100),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
		ManagedBy:     billing.ManuallyManagedLine,
	})
	line.TaxConfig = &productcatalog.TaxConfig{
		Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_99999999"},
	}

	result, err := s.BillingService.SimulateInvoice(ctx, billing.SimulateInvoiceInput{
		Namespace:  ns,
		CustomerID: &cust.ID,
		Currency:   currencyx.Code(currency.USD),
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

// ── Group E: Profile.Merge TaxCodeID propagation ─────────────────────────

// E1: When both profile and override have Stripe codes, MergeTaxConfigs gives the profile's
// code precedence (profile is the second/overrides arg in the call site). Both TaxCode
// entities are created; the merged config references the profile's entity.
func (s *TaxCodeDualWriteTestSuite) TestProfileMergeBothCodeProfileWins() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}))
	cust := s.CreateTestCustomer(ns, "test")

	_, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		Invoicing: billing.InvoicingOverrideConfig{
			DefaultTaxConfig: &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
			},
		},
	})
	s.NoError(err)

	override, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{Namespace: ns, ID: cust.ID},
	})
	s.NoError(err)

	cfg := override.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig
	// MergeTaxConfigs(override, profile) — profile is the second/"overrides" argument and
	// wins field-by-field over the customer override when both fields are non-nil.
	s.assertTaxConfigHasStripeCode(cfg, "txcd_10000000")

	// Both Stripe codes create independent TaxCode entities.
	count, err := s.DBClient.TaxCode.Query().
		Where(taxcodedb.Namespace(ns)).
		Count(ctx)
	s.NoError(err)
	s.Equal(2, count, "both Stripe codes should have a TaxCode entity")
}

// E2: When override has only behavior (no Stripe), the merged result inherits the profile's
// Stripe code and TaxCodeID while using the override's behavior.
func (s *TaxCodeDualWriteTestSuite) TestProfileMergeFieldByField() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}
	}))
	cust := s.CreateTestCustomer(ns, "test")

	_, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		Invoicing: billing.InvoicingOverrideConfig{
			DefaultTaxConfig: &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			},
		},
	})
	s.NoError(err)

	override, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{Namespace: ns, ID: cust.ID},
	})
	s.NoError(err)

	cfg := override.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig
	// Stripe code and TaxCodeID come from the profile; behavior comes from the override.
	s.assertTaxConfigHasStripeCode(cfg, "txcd_10000000")
	s.Require().NotNil(cfg.Behavior)
	s.Equal(productcatalog.ExclusiveTaxBehavior, *cfg.Behavior)
}

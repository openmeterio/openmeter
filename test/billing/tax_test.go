package billing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingconvert "github.com/openmeterio/openmeter/openmeter/billing/apiconvert"
	billinghttpdriver "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/customer"
	billinginvoicelinedb "github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
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
	profileTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
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
		selectedTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_20000000")
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
		s.ErrorIs(err, billing.ErrTaxCodeNotFound)

		// let's update the invoice
		editedTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_30000000")
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
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
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

		// v1 shows the tax code's current mapping for the default and the lines.
		taxCodes, err := billingconvert.ResolveInvoiceTaxCodes(ctx, s.TaxCodeResolver, billingconvert.ResolveInvoiceTaxCodesInput{
			Namespace:     namespace,
			Workflows:     []billing.WorkflowConfig{reread.Workflow.Config},
			StandardLines: reread.Lines.OrEmpty(),
		})
		s.Require().NoError(err)

		apiInvoice, err := billinghttpdriver.MapStandardInvoiceToAPI(reread, taxCodes)
		s.Require().NoError(err)
		s.Require().NotNil(apiInvoice.Workflow.Workflow.Invoicing)
		s.Equal(&apiv1models.TaxConfig{
			Behavior:  lo.ToPtr(apiv1models.TaxBehaviorInclusive),
			Stripe:    &apiv1models.StripeTaxConfig{Code: "txcd_20000000"},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		}, apiInvoice.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	})
}

// A line's stored stripe.code is an input echo, not the source of truth: v1 output renders the
// Stripe code of the Tax Code the line's TaxCodeID references at read time, so a Tax Code that
// was remapped after the invoice was created shows the current mapping.
func (s *InvoicingTaxTestSuite) TestInvoiceLineRendersCurrentTaxCodeMapping() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-tax-line-remap")

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	cust := s.CreateTestCustomer(namespace, "test")
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")

	// given: a draft invoice whose line references the tax code with its original mapping
	invoice := s.generateDraftInvoiceWithTaxCodeID(ctx, cust, taxCode.ID)

	// when: the tax code is remapped to another Stripe code
	_, err := s.TaxCodeEnv.Service.UpdateTaxCode(ctx, taxcode.UpdateTaxCodeInput{
		NamespacedID: taxCode.NamespacedID,
		Name:         taxCode.Name,
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_50000000"},
		},
		// The test drives the update directly, not through the API, so the immutability
		// guard doesn't apply.
		AllowAppMappingOverride: true,
	})
	s.Require().NoError(err)

	// then: the line renders the current mapping, not the stored one
	reread, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)

	taxCodes, err := billingconvert.ResolveInvoiceTaxCodes(ctx, s.TaxCodeResolver, billingconvert.ResolveInvoiceTaxCodesInput{
		Namespace:     namespace,
		Workflows:     []billing.WorkflowConfig{reread.Workflow.Config},
		StandardLines: reread.Lines.OrEmpty(),
	})
	s.Require().NoError(err)

	apiInvoice, err := billinghttpdriver.MapStandardInvoiceToAPI(reread, taxCodes)
	s.Require().NoError(err)
	s.Require().NotNil(apiInvoice.Lines)
	s.Require().Len(*apiInvoice.Lines, 1)
	s.Require().NotNil((*apiInvoice.Lines)[0].TaxConfig)
	s.Equal(&apiv1models.TaxConfig{
		Behavior:  lo.ToPtr(apiv1models.TaxBehaviorExclusive),
		Stripe:    &apiv1models.StripeTaxConfig{Code: "txcd_50000000"},
		TaxCodeId: lo.ToPtr(taxCode.ID),
	}, (*apiInvoice.Lines)[0].TaxConfig)
}

// generateDraftInvoiceWithTaxCodeID creates a draft invoice whose lines explicitly reference
// the given Tax Code ID.
func (s *InvoicingTaxTestSuite) generateDraftInvoiceWithTaxCodeID(ctx context.Context, cust *customer.Customer, taxCodeID string) billing.StandardInvoice {
	invoice := s.generateDraftInvoice(ctx, cust)

	// Stamping the line's tax config through the calculator leaves the inherited default in
	// place; give the line its own reference so the read side resolves it as a line ID.
	updated, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      invoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceSystem,
		EditFn: func(invoice *billing.StandardInvoice) error {
			lines, err := invoice.Lines.MapWithErr(func(line *billing.StandardLine) (*billing.StandardLine, error) {
				line.TaxConfig = &billing.TaxConfig{
					TaxConfig: productcatalog.TaxConfig{
						Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
						TaxCodeID: lo.ToPtr(taxCodeID),
					},
				}

				return line, nil
			})
			if err != nil {
				return err
			}

			invoice.Lines = lines

			return nil
		},
	})
	s.Require().NoError(err)

	return updated
}

// A deleted profile default is inherited by the invoices created from it. Calculation stamps the
// deleted Tax Code itself onto the lines by ID: it must not look the default up by its Stripe
// mapping, which would create a new Tax Code and attach it to lines pointing at the deleted one.
func (s *InvoicingTaxTestSuite) TestDeletedDefaultTaxCodeIsCopiedIntoInvoices() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-tax-default-deleted")

	// given: a profile default whose tax code was deleted afterwards, and no live tax code with
	// its Stripe mapping
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	cust := s.CreateTestCustomer(namespace, "test")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	profile := s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	deleted := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{TaxCodeID: deleted.ID})
	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: deleted.NamespacedID}))

	taxCodesBefore := s.countTaxCodes(ctx, namespace)

	var invoice billing.StandardInvoice

	s.Run("when: an invoice is created from the profile", func() {
		invoice = s.generateDraftInvoice(ctx, cust)
	})

	s.Run("then: the invoice inherits the deleted default and its lines reference it", func() {
		s.Equal(deleted.ID, invoice.Workflow.Config.Invoicing.DefaultTaxCodeID())
		s.requireLinesReferenceTaxCode(invoice, deleted.ID)

		lines := invoice.Lines.OrEmpty()
		s.Equal(&productcatalog.StripeTaxConfig{Code: "txcd_10000000"}, lines[0].TaxConfig.Stripe)

		s.Equal(taxCodesBefore, s.countTaxCodes(ctx, namespace), "calculation must not create a tax code")
	})

	s.Run("then: the reloaded invoice and the stored rows reference the deleted tax code", func() {
		reloaded, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.requireLinesReferenceTaxCode(reloaded, deleted.ID)
		s.requirePersistedLinesReferenceTaxCode(ctx, invoice, deleted.ID)
	})

	// A system edit recalculates without the API edit diff, which allows an edit that
	// leaves the lines' Tax Code references untouched.
	s.Run("when: the invoice is recalculated without touching the default", func() {
		edited, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      invoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceSystem,
			EditFn: func(invoice *billing.StandardInvoice) error {
				invoice.Description = lo.ToPtr("recalculated")

				return nil
			},
		})

		// then: the inherited deleted default passes and recalculation creates no tax code either
		s.Require().NoError(err)
		s.requireLinesReferenceTaxCode(edited, deleted.ID)
		s.requirePersistedLinesReferenceTaxCode(ctx, edited, deleted.ID)
		s.Equal(taxCodesBefore, s.countTaxCodes(ctx, namespace), "recalculation must not create a tax code")
	})

	s.Run("when: an API edit touches another field of a line holding the deleted tax code", func() {
		edited, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      invoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.StandardInvoice) error {
				line := invoice.Lines.OrEmpty()[0]
				line.Name = "edited name"
				if !invoice.Lines.ReplaceByID(line.ID, line) {
					return fmt.Errorf("line %s not found", line.ID)
				}

				return nil
			},
		})

		// then: the unchanged inherited deleted reference passes
		s.Require().NoError(err)
		s.requireLinesReferenceTaxCode(edited, deleted.ID)
	})

	s.Run("when: an edit sets the default to another deleted tax code", func() {
		otherDeleted := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_20000000")
		s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: otherDeleted.NamespacedID}))

		_, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      invoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.StandardInvoice) error {
				invoice.Workflow.Config.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{TaxCodeID: otherDeleted.ID}

				return nil
			},
		})

		// then: the authored deleted reference is rejected
		s.ErrorIs(err, billing.ErrTaxCodeDeleted)
	})
}

// API edits on gathering invoices apply the same authored/inherited split as on standard
// invoices: touching another field of a line holding a deleted Tax Code passes, while pointing
// a line at a deleted code is rejected.
func (s *InvoicingTaxTestSuite) TestGatheringInvoiceEditTaxCodeAuthoredVsInherited() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-tax-gathering-edit-deleted")

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	cust := s.CreateTestCustomer(namespace, "test")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	deleted := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: deleted.NamespacedID}))

	now := time.Now().Truncate(time.Microsecond).In(time.UTC)
	res, err := s.BillingService.CreatePendingInvoiceLines(ctx, s.newTaxedPendingLinesInput(namespace, cust.GetID(), now, &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(deleted.ID)}, billing.ChangeSourceSystem))
	s.Require().NoError(err)
	s.Require().Len(res.Lines, 1)

	s.Run("when: an API edit touches another field of a line holding the deleted tax code", func() {
		edited, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      res.Invoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				line, ok := invoice.Lines.GetByID(res.Lines[0].ID)
				if !ok {
					return fmt.Errorf("line[%s] not found", res.Lines[0].ID)
				}

				line.Name = "edited gathering line"

				return invoice.Lines.ReplaceByID(line)
			},
		})

		// then: the unchanged inherited deleted reference passes
		s.Require().NoError(err)
		editedLine, ok := edited.Lines.GetByID(res.Lines[0].ID)
		s.Require().True(ok)
		s.Require().NotNil(editedLine.TaxConfig)
		s.Equal(lo.ToPtr(deleted.ID), editedLine.TaxConfig.TaxCodeID)
	})

	s.Run("when: an API edit repoints the line at another deleted tax code", func() {
		// The line already holds this test's deleted code, so a same-ID edit is an unchanged
		// inherited reference; repointing at a different deleted code is authored.
		otherDeleted := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_20000000")
		s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: otherDeleted.NamespacedID}))

		_, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      res.Invoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				line, ok := invoice.Lines.GetByID(res.Lines[0].ID)
				if !ok {
					return fmt.Errorf("line[%s] not found", res.Lines[0].ID)
				}

				line.TaxConfig = &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(otherDeleted.ID)}

				return invoice.Lines.ReplaceByID(line)
			},
		})

		// then: the authored deleted reference is rejected
		s.ErrorIs(err, billing.ErrTaxCodeDeleted)
	})

	s.Run("when: an API edit repoints the line at a live tax code the change persists", func() {
		live := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_30000000")

		edited, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      res.Invoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				line, ok := invoice.Lines.GetByID(res.Lines[0].ID)
				if !ok {
					return fmt.Errorf("line[%s] not found", res.Lines[0].ID)
				}

				line.TaxConfig = &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(live.ID)}

				return invoice.Lines.ReplaceByID(line)
			},
		})

		// then: the authored live reference is accepted and persisted
		s.Require().NoError(err)
		editedLine, ok := edited.Lines.GetByID(res.Lines[0].ID)
		s.Require().True(ok)
		s.Require().NotNil(editedLine.TaxConfig)
		s.Equal(lo.ToPtr(live.ID), editedLine.TaxConfig.TaxCodeID)
	})
}

// After the 2026-06 dedupe, a deleted Tax Code can share its Stripe mapping with a live one.
// Lines inheriting the deleted default must keep the deleted Tax Code, not get the live one.
func (s *InvoicingTaxTestSuite) TestDeletedDefaultTaxCodeIsNotSwappedForLiveMapping() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-tax-default-deleted-remapped")

	// given: a deleted profile default and a live tax code with the same Stripe mapping
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	cust := s.CreateTestCustomer(namespace, "test")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	profile := s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	deleted := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{TaxCodeID: deleted.ID})
	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: deleted.NamespacedID}))

	live := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	s.Require().NotEqual(deleted.ID, live.ID)

	// when: an invoice is created from the profile
	invoice := s.generateDraftInvoice(ctx, cust)

	// then: its lines keep the deleted tax code, also in the stored snapshot
	s.requireLinesReferenceTaxCode(invoice, deleted.ID)
	s.requirePersistedLinesReferenceTaxCode(ctx, invoice, deleted.ID)
}

// API callers author the Tax Code IDs they send on pending lines: a deleted one is rejected
// with the deleted-specific error, a missing one with not-found. A system-source create
// carrying the same reference is inherited and passes.
func (s *InvoicingTaxTestSuite) TestPendingLinesTaxCodeAuthoredVsInherited() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-tax-pending-lines-deleted")

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	cust := s.CreateTestCustomer(namespace, "test")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	deleted := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: deleted.NamespacedID}))

	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	s.Run("when: an API create references the deleted tax code", func() {
		input := s.newTaxedPendingLinesInput(namespace, cust.GetID(), now, &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(deleted.ID)}, billing.ChangeSourceAPIRequest)

		_, err := s.BillingService.CreatePendingInvoiceLines(ctx, input)

		// then: it is rejected as deleted
		s.Require().Error(err)
		s.ErrorIs(err, billing.ErrTaxCodeDeleted)
		s.True(billing.IsValidationIssueOnly(err), "expected a validation error, got: %v", err)
	})

	s.Run("when: an API create references a missing tax code", func() {
		input := s.newTaxedPendingLinesInput(namespace, cust.GetID(), now, &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(ulid.Make().String())}, billing.ChangeSourceAPIRequest)

		_, err := s.BillingService.CreatePendingInvoiceLines(ctx, input)

		// then: it is rejected as not found
		s.Require().Error(err)
		s.ErrorIs(err, billing.ErrTaxCodeNotFound)
	})

	s.Run("when: a system create references the deleted tax code", func() {
		input := s.newTaxedPendingLinesInput(namespace, cust.GetID(), now, &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(deleted.ID)}, billing.ChangeSourceSystem)

		res, err := s.BillingService.CreatePendingInvoiceLines(ctx, input)

		// then: the inherited reference passes and the line keeps the deleted tax code
		s.Require().NoError(err)
		s.Require().Len(res.Lines, 1)
		s.Require().NotNil(res.Lines[0].TaxConfig)
		s.Equal(lo.ToPtr(deleted.ID), res.Lines[0].TaxConfig.TaxCodeID)
	})
}

// newTaxedPendingLinesInput builds a one-line pending-lines create request whose line carries
// the given tax config.
func (s *InvoicingTaxTestSuite) newTaxedPendingLinesInput(namespace string, customerID customer.CustomerID, now time.Time, taxConfig *productcatalog.TaxConfig, changeSource billing.ChangeSource) billing.CreatePendingInvoiceLinesInput {
	line := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     namespace,
		Period:        timeutil.ClosedPeriod{From: now, To: now.Add(24 * time.Hour)},
		InvoiceAt:     now,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "deleted tax line",
		PerUnitAmount: alpacadecimal.NewFromFloat(100),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	line.TaxConfig = taxConfig

	return billing.CreatePendingInvoiceLinesInput{
		Customer:     customerID,
		Currency:     currencyx.FiatCode(currency.USD),
		Lines:        billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{line}),
		ChangeSource: changeSource,
	}
}

// Calculation loads a line's Tax Code by its ID, deleted included, and never creates a Tax
// Code for a line referencing a deleted one.
func (s *InvoicingTaxTestSuite) TestCalculationResolvesDeletedTaxCodeByID() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-tax-calc-deleted")

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	cust := s.CreateTestCustomer(namespace, "test")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	deleted := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: deleted.NamespacedID}))

	taxCodesBefore := s.countTaxCodes(ctx, namespace)

	now := time.Now().Truncate(time.Microsecond).In(time.UTC)
	deletedLine := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     namespace,
		Period:        timeutil.ClosedPeriod{From: now, To: now.Add(24 * time.Hour)},
		InvoiceAt:     now,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "deleted tax line",
		PerUnitAmount: alpacadecimal.NewFromFloat(100),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	deletedLine.TaxConfig = &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(deleted.ID)}
	res, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer:     cust.GetID(),
		Currency:     currencyx.FiatCode(currency.USD),
		Lines:        billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{deletedLine}),
		ChangeSource: billing.ChangeSourceSystem,
	})
	s.Require().NoError(err)
	s.Require().Len(res.Lines, 1)

	// when: the pending line is collected into a standard invoice
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)

	// then: the line keeps the deleted ID with the deleted Tax Code attached, and no Tax
	// Code was created for the line's stale Stripe mapping
	invoice := invoices[0]
	s.requireLinesReferenceTaxCode(invoice, deleted.ID)
	s.requirePersistedLinesReferenceTaxCode(ctx, invoice, deleted.ID)
	s.Equal(taxCodesBefore, s.countTaxCodes(ctx, namespace), "calculation must not create a tax code")
}

// requireLinesReferenceTaxCode requires every line to point at the Tax Code by ID and carry that
// same Tax Code, the rule the adapter applies on read.
func (s *InvoicingTaxTestSuite) requireLinesReferenceTaxCode(invoice billing.StandardInvoice, taxCodeID string) {
	s.T().Helper()

	lines := invoice.Lines.OrEmpty()
	s.Require().NotEmpty(lines)

	for _, line := range lines {
		s.Require().NotNil(line.TaxConfig)
		s.Equal(lo.ToPtr(taxCodeID), line.TaxConfig.TaxCodeID)
		s.Require().NotNil(line.TaxConfig.TaxCode)
		s.Equal(taxCodeID, line.TaxConfig.TaxCode.ID)
	}
}

// requirePersistedLinesReferenceTaxCode checks the stored rows directly: the service returns lines
// re-read through the adapter, which backfills the Tax Code entity from the tax_code_id edge, so
// only the row shows what was persisted. The tax_config JSONB snapshot carries the resolved entity
// only when a write happened after calculation stamped it; a line that references its own Tax Code
// ID is first persisted before that stamping and later recalculations leave the JSON untouched, so
// the entity is asserted only when present.
func (s *InvoicingTaxTestSuite) requirePersistedLinesReferenceTaxCode(ctx context.Context, invoice billing.StandardInvoice, taxCodeID string) {
	s.T().Helper()

	for _, line := range invoice.Lines.OrEmpty() {
		dbLine, err := s.DBClient.BillingInvoiceLine.Query().
			Where(billinginvoicelinedb.ID(line.ID)).
			Only(ctx)
		s.Require().NoError(err)

		s.Equal(lo.ToPtr(taxCodeID), dbLine.TaxCodeID)
		s.Require().NotNil(dbLine.TaxConfig)
		s.Equal(lo.ToPtr(taxCodeID), dbLine.TaxConfig.TaxCodeID)

		if dbLine.TaxConfig.TaxCode != nil {
			s.Equal(taxCodeID, dbLine.TaxConfig.TaxCode.ID)
		}
	}
}

// countTaxCodes counts the namespace's Tax Codes, deleted ones included.
func (s *InvoicingTaxTestSuite) countTaxCodes(ctx context.Context, namespace string) int {
	s.T().Helper()

	count, err := s.DBClient.TaxCode.Query().Where(taxcodedb.Namespace(namespace)).Count(ctx)
	s.Require().NoError(err)

	return count
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
			ChangeSource: billing.ChangeSourceAPIRequest,
			Customer:     customer.GetID(),
			Currency:     currencyx.FiatCode(currency.USD),
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
			ChangeSource: billing.ChangeSourceAPIRequest,
			Customer:     customer.GetID(),
			Currency:     currencyx.FiatCode(currency.USD),
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

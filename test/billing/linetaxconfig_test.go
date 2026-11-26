package billing

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	billinginvoicelinedb "github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// LineTaxConfigTestSuite covers the legacy tax_config JSON column of invoice lines. The domain
// only carries the Tax Code ID and behavior (tax_code_id/tax_behavior columns); tax_config is
// frozen: never written for new lines, never changed on existing ones, and only read for the
// legacy Stripe code of unstamped lines.
type LineTaxConfigTestSuite struct {
	BaseSuite
}

func TestLineTaxConfig(t *testing.T) {
	suite.Run(t, new(LineTaxConfigTestSuite))
}

// createPendingLines creates one flat fee pending line per entry, named by the map key and
// carrying the given tax config.
func (s *LineTaxConfigTestSuite) createPendingLines(namespace string, cust *customer.Customer, now time.Time, lines map[string]*productcatalog.TaxCodeConfig) *billing.CreatePendingInvoiceLinesResult {
	s.T().Helper()

	gatheringLines := make([]billing.GatheringLine, 0, len(lines))
	for name, taxConfig := range lines {
		line := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
			Namespace:     namespace,
			Period:        timeutil.ClosedPeriod{From: now, To: now.Add(24 * time.Hour)},
			InvoiceAt:     now,
			ManagedBy:     billing.ManuallyManagedLine,
			Name:          name,
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			PaymentTerm:   productcatalog.InAdvancePaymentTerm,
		})
		line.TaxConfig = taxConfig

		gatheringLines = append(gatheringLines, line)
	}

	res, err := s.BillingService.CreatePendingInvoiceLines(s.T().Context(), billing.CreatePendingInvoiceLinesInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
		Customer:     cust.GetID(),
		Currency:     currencyx.FiatCode(currency.USD),
		Lines:        billing.NewCreatePendingInvoiceLines(gatheringLines),
	})
	s.Require().NoError(err)
	s.Require().Len(res.Lines, len(lines))

	return res
}

// storedLine reads the persisted invoice line row.
func (s *LineTaxConfigTestSuite) storedLine(lineID string) *db.BillingInvoiceLine {
	s.T().Helper()

	row, err := s.DBClient.BillingInvoiceLine.Query().Where(billinginvoicelinedb.ID(lineID)).Only(s.T().Context())
	s.Require().NoError(err)

	return row
}

// storedTaxConfigJSON returns the persisted tax_config snapshot of a line as JSON.
func (s *LineTaxConfigTestSuite) storedTaxConfigJSON(lineID string) string {
	s.T().Helper()

	raw, err := json.Marshal(s.storedLine(lineID).TaxConfig)
	s.Require().NoError(err)

	return string(raw)
}

// requireNullTaxConfig requires the line's tax_config to be SQL NULL. The ent field is a value
// type, so NULL and an empty object both read as the zero value; the predicate tells them apart.
func (s *LineTaxConfigTestSuite) requireNullTaxConfig(lineID string) {
	s.T().Helper()

	isNull, err := s.DBClient.BillingInvoiceLine.Query().
		Where(billinginvoicelinedb.ID(lineID), billinginvoicelinedb.TaxConfigIsNil()).
		Exist(s.T().Context())
	s.Require().NoError(err)
	s.True(isNull, "tax_config must be NULL")
}

// seedLegacyTaxConfig writes a tax_config JSON straight into the line row, the way lines from
// before the column was frozen look.
func (s *LineTaxConfigTestSuite) seedLegacyTaxConfig(lineID string, legacy billing.StoredLineTaxConfig) {
	s.T().Helper()

	s.Require().NoError(s.DBClient.BillingInvoiceLine.UpdateOneID(lineID).SetTaxConfig(legacy).Exec(s.T().Context()))
}

// TestNewLinesAreCreatedWithoutTaxConfig checks that new gathering and standard lines keep the
// tax identity in the tax_code_id/tax_behavior columns and leave the legacy tax_config NULL.
func (s *LineTaxConfigTestSuite) TestNewLinesAreCreatedWithoutTaxConfig() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-no-legacy-write")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: a customer, a Tax Code with a Stripe mapping and one without
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	cust := s.CreateTestCustomer(namespace, "test")
	mappedTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	unmappedTaxCode := s.TaxCodeEnv.CreateTaxCode(s.T(), namespace)

	var pending *billing.CreatePendingInvoiceLinesResult

	s.Run("when: pending lines are created", func() {
		pending = s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{
			"mapped":        {Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior), TaxCodeID: mappedTaxCode.ID},
			"unmapped":      {TaxCodeID: unmappedTaxCode.ID},
			"behavior-only": {Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)},
		})
	})

	s.Run("then: gathering lines have the columns set and no tax_config", func() {
		gatheringByName := lo.KeyBy(pending.Lines, func(line billing.GatheringLine) string { return line.Name })

		mapped := s.storedLine(gatheringByName["mapped"].ID)
		s.Equal(mappedTaxCode.ID, lo.FromPtr(mapped.TaxCodeID))
		s.Equal(productcatalog.ExclusiveTaxBehavior, lo.FromPtr(mapped.TaxBehavior))
		s.requireNullTaxConfig(mapped.ID)

		unmapped := s.storedLine(gatheringByName["unmapped"].ID)
		s.Equal(unmappedTaxCode.ID, lo.FromPtr(unmapped.TaxCodeID))
		s.Nil(unmapped.TaxBehavior)
		s.requireNullTaxConfig(unmapped.ID)

		behaviorOnly := s.storedLine(gatheringByName["behavior-only"].ID)
		s.Nil(behaviorOnly.TaxCodeID)
		s.Equal(productcatalog.InclusiveTaxBehavior, lo.FromPtr(behaviorOnly.TaxBehavior))
		s.requireNullTaxConfig(behaviorOnly.ID)
	})

	s.Run("then: standard lines have the columns set and no tax_config", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     &now,
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)

		standardByName := lo.KeyBy(invoices[0].Lines.OrEmpty(), func(line *billing.StandardLine) string { return line.Name })
		s.Require().Len(standardByName, 3)

		mapped := s.storedLine(standardByName["mapped"].ID)
		s.Equal(mappedTaxCode.ID, lo.FromPtr(mapped.TaxCodeID))
		s.Equal(productcatalog.ExclusiveTaxBehavior, lo.FromPtr(mapped.TaxBehavior))
		s.requireNullTaxConfig(mapped.ID)

		unmapped := s.storedLine(standardByName["unmapped"].ID)
		s.Equal(unmappedTaxCode.ID, lo.FromPtr(unmapped.TaxCodeID))
		s.Nil(unmapped.TaxBehavior)
		s.requireNullTaxConfig(unmapped.ID)

		behaviorOnly := s.storedLine(standardByName["behavior-only"].ID)
		s.Nil(behaviorOnly.TaxCodeID)
		s.Equal(productcatalog.InclusiveTaxBehavior, lo.FromPtr(behaviorOnly.TaxBehavior))
		s.requireNullTaxConfig(behaviorOnly.ID)
	})
}

// TestStoredTaxConfigKeptOnLinePastDraftUpsert checks that a legacy tax_config stays as it is when
// the line is re-upserted for a reason unrelated to tax (here an invoicing app merging line
// external IDs on the draft -> issuing transition), even if it no longer matches what the current
// Tax Code would render.
func (s *LineTaxConfigTestSuite) TestStoredTaxConfigKeptOnLinePastDraftUpsert() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-legacy-kept-on-upsert")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: an invoice waiting in draft.syncing on a custom invoicing app with sync hooks
	customInvoicing := s.SetupCustomInvoicing(namespace, WithCustomInvoicingConfig(appcustominvoicing.Configuration{
		EnableDraftSyncHook:   true,
		EnableIssuingSyncHook: true,
	}))
	s.ProvisionBillingProfile(ctx, namespace, customInvoicing.App.GetID(), WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
		profile.WorkflowConfig.Invoicing.DraftPeriod = lo.Must(datetime.ISODurationString("P0D").Parse())
	}))
	cust := s.CreateTestCustomer(namespace, "test")
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")

	s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{
		"taxed": {Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior), TaxCodeID: taxCode.ID},
	})

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Equal(billing.StandardInvoiceStatusDraftSyncing, invoices[0].Status)

	invoice := invoices[0]
	s.Require().Len(invoice.Lines.OrEmpty(), 1)
	lineID := invoice.Lines.OrEmpty()[0].ID

	// and: the line has a legacy tax_config from before a remap of the Tax Code (older stripe
	// code and Tax Code state), with tax_code_id and tax_behavior matching it
	legacySnapshot := billing.StoredLineTaxConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(taxCode.ID),
		Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_00000001"},
		TaxCode: &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: taxCode.ID},
			Key:          taxCode.Key,
			Name:         "legacy tax code name",
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_00000001"},
			},
		},
	}
	s.seedLegacyTaxConfig(lineID, legacySnapshot)

	legacyJSON := s.storedTaxConfigJSON(lineID)
	s.Contains(legacyJSON, "txcd_00000001", "precondition: the legacy snapshot is stored")

	// when: the app syncs the draft, which merges the line external ID and moves the invoice past draft
	synced, err := s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{
		InvoiceID: invoice.GetInvoiceID(),
		UpsertInvoiceResults: billing.NewUpsertStandardInvoiceResult().
			SetInvoiceNumber("DRAFT-1").
			SetExternalID("ext-invoice").
			AddLineExternalID(lineID, "ext-line"),
	})
	s.Require().NoError(err)

	// then: the invoice left draft and the line was re-upserted (it carries the external ID)
	s.Equal(billing.StandardInvoiceStatusIssuingSyncing, synced.Status)
	s.Require().Len(synced.Lines.OrEmpty(), 1)
	s.Equal("ext-line", synced.Lines.OrEmpty()[0].ExternalIDs.Invoicing)

	// and: the legacy tax_config is untouched, the columns still hold the tax identity
	s.Equal(legacyJSON, s.storedTaxConfigJSON(lineID))

	row := s.storedLine(lineID)
	s.Equal(taxCode.ID, lo.FromPtr(row.TaxCodeID))
	s.Equal(productcatalog.ExclusiveTaxBehavior, lo.FromPtr(row.TaxBehavior))
}

// TestDraftLineWithDeletedTaxCodeCanBeResaved checks that a draft line inheriting a soft-deleted
// Tax Code still passes calculation and upsert and keeps referencing it.
func (s *LineTaxConfigTestSuite) TestDraftLineWithDeletedTaxCodeCanBeResaved() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-deleted-resave")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: a pending line referencing a Tax Code that is deleted before the line is collected
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	cust := s.CreateTestCustomer(namespace, "test")
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")

	s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{
		"taxed": {Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior), TaxCodeID: taxCode.ID},
	})
	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: taxCode.NamespacedID}))

	var invoice billing.StandardInvoice

	s.Run("when: the line is collected into a draft invoice", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     &now,
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		invoice = invoices[0]
	})

	s.Run("then: the line row references the deleted Tax Code", func() {
		row := s.storedLine(invoice.Lines.OrEmpty()[0].ID)
		s.Equal(taxCode.ID, lo.FromPtr(row.TaxCodeID))
		s.requireNullTaxConfig(row.ID)
	})

	s.Run("when: the draft line is edited and re-saved", func() {
		edited, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      invoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceSystem,
			EditFn: func(invoice *billing.StandardInvoice) error {
				for _, line := range invoice.Lines.OrEmpty() {
					line.Name = "edited name"
				}

				return nil
			},
		})

		// then: calculation and upsert succeed and the line still references the deleted Tax Code
		s.Require().NoError(err)
		s.Require().Len(edited.Lines.OrEmpty(), 1)
		s.Equal("edited name", edited.Lines.OrEmpty()[0].Name)
		s.Equal(taxCode.ID, edited.Lines.OrEmpty()[0].TaxConfig.TaxCodeID)

		row := s.storedLine(edited.Lines.OrEmpty()[0].ID)
		s.Equal(taxCode.ID, lo.FromPtr(row.TaxCodeID))
		s.requireNullTaxConfig(row.ID)
	})
}

// TestLegacyTaxConfigKeptWhenTaxCodeChanges checks that editing the Tax Code of a line that has a
// legacy tax_config (a draft standard line and a gathering line) changes the columns but leaves
// the JSON as it was, and that the legacy Stripe code is still read from it.
func (s *LineTaxConfigTestSuite) TestLegacyTaxConfigKeptWhenTaxCodeChanges() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-legacy-kept")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: a gathering line and a draft standard line, both with a legacy tax_config for the
	// original Tax Code
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	cust := s.CreateTestCustomer(namespace, "test")
	originalTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	otherTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_20000000")

	legacy := billing.StoredLineTaxConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(originalTaxCode.ID),
		Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
	}
	original := &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior), TaxCodeID: originalTaxCode.ID}

	// the first line is collected into a draft invoice, the second one stays gathering
	collected := s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{"collected": original})
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(collected.Lines, 1)

	draftInvoiceID := invoices[0].GetInvoiceID()
	draftLineID := invoices[0].Lines.OrEmpty()[0].ID

	gathering := s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{"gathering": original})
	gatheringLine := gathering.Lines[0]

	s.seedLegacyTaxConfig(draftLineID, legacy)
	s.seedLegacyTaxConfig(gatheringLine.ID, legacy)

	legacyJSON, err := json.Marshal(legacy)
	s.Require().NoError(err)
	s.Require().JSONEq(string(legacyJSON), s.storedTaxConfigJSON(draftLineID), "precondition: the legacy tax_config is stored")

	s.Run("when: the Tax Code of the draft standard line is changed", func() {
		_, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      draftInvoiceID,
			ChangeSource: billing.ChangeSourceSystem,
			EditFn: func(invoice *billing.StandardInvoice) error {
				for _, line := range invoice.Lines.OrEmpty() {
					line.TaxConfig = &productcatalog.TaxCodeConfig{Behavior: line.TaxConfig.Behavior, TaxCodeID: otherTaxCode.ID}
				}

				return nil
			},
		})
		s.Require().NoError(err)
	})

	s.Run("then: the column changed and the legacy tax_config did not", func() {
		s.Equal(otherTaxCode.ID, lo.FromPtr(s.storedLine(draftLineID).TaxCodeID))
		s.JSONEq(string(legacyJSON), s.storedTaxConfigJSON(draftLineID))
	})

	s.Run("then: the legacy Stripe code is still read from it", func() {
		s.Equal("txcd_10000000", s.readStandardLine(draftInvoiceID, draftLineID).LegacyStripeTaxCode)
	})

	s.Run("when: the Tax Code of the gathering line is changed", func() {
		_, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      billing.InvoiceID{Namespace: namespace, ID: gatheringLine.InvoiceID},
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				edited, ok := invoice.Lines.GetByID(gatheringLine.ID)
				s.Require().True(ok)

				edited.TaxConfig = &productcatalog.TaxCodeConfig{Behavior: edited.TaxConfig.Behavior, TaxCodeID: otherTaxCode.ID}

				return invoice.Lines.ReplaceByID(edited)
			},
		})
		s.Require().NoError(err)
	})

	s.Run("then: the column changed and the legacy tax_config did not", func() {
		s.Equal(otherTaxCode.ID, lo.FromPtr(s.storedLine(gatheringLine.ID).TaxCodeID))
		s.JSONEq(string(legacyJSON), s.storedTaxConfigJSON(gatheringLine.ID))
	})
}

// TestInvoicingAppTaxCodeRoundTrip checks that a standard line reads back the invoicing app tax code
// stamp as stored (a code, or "" for "stamped, provider default"), that a line that was never
// stamped reads NULL, and that its legacy stored stripe code is exposed for the unstamped fallback.
func (s *LineTaxConfigTestSuite) TestInvoicingAppTaxCodeRoundTrip() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-app-code-roundtrip")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: a draft invoice whose standard line is a legacy one: a stored stripe code, no stamp
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	cust := s.CreateTestCustomer(namespace, "test")
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")

	s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{
		"taxed": {Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior), TaxCodeID: taxCode.ID},
	})

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)

	invoiceID := invoices[0].GetInvoiceID()
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)
	lineID := invoices[0].Lines.OrEmpty()[0].ID

	s.Run("given: a line from before stamping, with a NULL stamp and a legacy stripe code", func() {
		s.setStoredInvoicingAppTaxCode(lineID, nil)
		s.seedLegacyTaxConfig(lineID, billing.StoredLineTaxConfig{
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: lo.ToPtr(taxCode.ID),
			Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		})
	})

	s.Run("then: the unstamped line reads NULL and exposes the legacy stripe code", func() {
		s.Nil(s.storedLine(lineID).InvoicingAppTaxCode)

		line := s.readStandardLine(invoiceID, lineID)
		s.Nil(line.InvoicingAppTaxCode)
		s.Equal("txcd_10000000", line.LegacyStripeTaxCode)
		s.Equal("txcd_10000000", line.BilledTaxCode(app.AppTypeStripe))
		s.Equal("", line.BilledTaxCode(app.AppTypeCustomInvoicing))
	})

	s.Run("when: the line is saved with a stamp", func() {
		s.setStoredInvoicingAppTaxCode(lineID, lo.ToPtr("txcd_20000000"))
	})

	s.Run("then: the stamp is persisted and read back, winning over the legacy code", func() {
		s.Equal("txcd_20000000", lo.FromPtr(s.storedLine(lineID).InvoicingAppTaxCode))

		line := s.readStandardLine(invoiceID, lineID)
		s.Equal("txcd_20000000", lo.FromPtr(line.InvoicingAppTaxCode))
		s.Equal("txcd_20000000", line.BilledTaxCode(app.AppTypeStripe))
		s.Equal("txcd_10000000", line.LegacyStripeTaxCode, "the stored legacy JSON is not rewritten")
	})

	s.Run("when: the line is saved with an empty stamp", func() {
		s.setStoredInvoicingAppTaxCode(lineID, lo.ToPtr(""))
	})

	s.Run("then: the empty stamp is kept apart from NULL and still wins over the legacy code", func() {
		stored := s.storedLine(lineID).InvoicingAppTaxCode
		s.Require().NotNil(stored)
		s.Equal("", *stored)

		line := s.readStandardLine(invoiceID, lineID)
		s.Require().NotNil(line.InvoicingAppTaxCode)
		s.Equal("", line.BilledTaxCode(app.AppTypeStripe))
	})
}

// TestGatheringLinesKeepInvoicingAppTaxCodeNull checks that gathering lines, which share the
// billing_invoice_lines table, never get an invoicing app tax code stamp, on insert or on a later
// upsert.
func (s *LineTaxConfigTestSuite) TestGatheringLinesKeepInvoicingAppTaxCodeNull() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-app-code-gathering")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: a pending gathering line
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	cust := s.CreateTestCustomer(namespace, "test")
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")

	pending := s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{
		"taxed": {TaxCodeID: taxCode.ID},
	})
	line := pending.Lines[0]

	s.Run("then: the insert leaves the column NULL", func() {
		s.Nil(s.storedLine(line.ID).InvoicingAppTaxCode)
	})

	s.Run("when: the gathering line is upserted again", func() {
		_, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      billing.InvoiceID{Namespace: namespace, ID: line.InvoiceID},
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				edited, ok := invoice.Lines.GetByID(line.ID)
				s.Require().True(ok)

				edited.Name = "edited"

				return invoice.Lines.ReplaceByID(edited)
			},
		})
		s.Require().NoError(err)
	})

	s.Run("then: the column is still NULL", func() {
		row := s.storedLine(line.ID)
		s.Equal("edited", row.Name)
		s.Nil(row.InvoicingAppTaxCode)
	})
}

// readStandardLine loads a standard line through the billing service.
func (s *LineTaxConfigTestSuite) readStandardLine(invoiceID billing.InvoiceID, lineID string) *billing.StandardLine {
	s.T().Helper()

	invoice, err := s.BillingService.GetStandardInvoiceById(s.T().Context(), billing.GetStandardInvoiceByIdInput{
		Invoice: invoiceID,
		Expand:  billing.StandardInvoiceExpands{billing.StandardInvoiceExpandLines},
	})
	s.Require().NoError(err)

	line := invoice.Lines.GetByID(lineID)
	s.Require().NotNil(line)

	return line
}

// setStoredInvoicingAppTaxCode writes the invoicing app tax code stamp straight into the line row
// (nil clears it to NULL). Going through the invoice update path would restamp the draft.
func (s *LineTaxConfigTestSuite) setStoredInvoicingAppTaxCode(lineID string, code *string) {
	s.T().Helper()

	update := s.DBClient.BillingInvoiceLine.UpdateOneID(lineID)
	if code == nil {
		update = update.ClearInvoicingAppTaxCode()
	} else {
		update = update.SetInvoicingAppTaxCode(*code)
	}

	s.Require().NoError(update.Exec(s.T().Context()))
}

// requireInvoicingAppTaxCode requires the persisted stamp of a line, and the stamp the billing
// service reads back, to be want (nil meaning NULL).
func (s *LineTaxConfigTestSuite) requireInvoicingAppTaxCode(invoiceID billing.InvoiceID, lineID string, want *string) {
	s.T().Helper()

	s.Equal(want, s.storedLine(lineID).InvoicingAppTaxCode, "stored stamp")
	s.Equal(want, s.readStandardLine(invoiceID, lineID).InvoicingAppTaxCode, "stamp read through the billing service")
}

// linesByName indexes an invoice's lines by name.
func linesByName(invoice billing.StandardInvoice) map[string]*billing.StandardLine {
	return lo.SliceToMap(invoice.Lines.OrEmpty(), func(line *billing.StandardLine) (string, *billing.StandardLine) {
		return line.Name, line
	})
}

// TestDraftInvoiceStampsInvoicingAppTaxCode checks that a draft created from pending lines stamps
// each line with the invoicing app's code for its Tax Code, or "" when there is none.
func (s *LineTaxConfigTestSuite) TestDraftInvoiceStampsInvoicingAppTaxCode() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-app-code-stamp")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: a sandbox invoicing app and pending lines with a mapped, an unmapped and no Tax Code
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	cust := s.CreateTestCustomer(namespace, "test")
	mapped := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeSandbox, "sandbox-code")
	stripeOnly := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")

	s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{
		"mapped":     {TaxCodeID: mapped.ID},
		"stripeOnly": {TaxCodeID: stripeOnly.ID},
		"none":       nil,
	})

	var invoice billing.StandardInvoice

	s.Run("when: the pending lines are collected into a draft invoice", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     &now,
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		invoice = invoices[0]
	})

	s.Run("then: every line carries the stamp of its Tax Code", func() {
		lines := linesByName(invoice)
		s.Require().Len(lines, 3)

		s.requireInvoicingAppTaxCode(invoice.GetInvoiceID(), lines["mapped"].ID, lo.ToPtr("sandbox-code"))
		s.requireInvoicingAppTaxCode(invoice.GetInvoiceID(), lines["stripeOnly"].ID, lo.ToPtr(""))
		s.requireInvoicingAppTaxCode(invoice.GetInvoiceID(), lines["none"].ID, lo.ToPtr(""))
	})

	s.Run("then: the invoice returned by the collection already has the stamps", func() {
		lines := linesByName(invoice)

		s.Equal("sandbox-code", lo.FromPtr(lines["mapped"].InvoicingAppTaxCode))
		s.Equal("sandbox-code", lines["mapped"].BilledTaxCode(app.AppTypeSandbox))
	})
}

// TestDraftRecalculationStampsUnstampedLine checks that a draft line from before stamping (NULL in
// the database) is stamped by the next draft calculation.
func (s *LineTaxConfigTestSuite) TestDraftRecalculationStampsUnstampedLine() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-app-code-restamp-null")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: a draft invoice whose line is mapped for the invoicing app but has a NULL stamp
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	cust := s.CreateTestCustomer(namespace, "test")
	mapped := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeSandbox, "sandbox-code")

	s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{
		"mapped": {TaxCodeID: mapped.ID},
	})

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)

	invoiceID := invoices[0].GetInvoiceID()
	lineID := linesByName(invoices[0])["mapped"].ID

	s.setStoredInvoicingAppTaxCode(lineID, nil)
	s.requireInvoicingAppTaxCode(invoiceID, lineID, nil)

	// when: the draft is edited, which recalculates it
	_, err = s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      invoiceID,
		ChangeSource: billing.ChangeSourceSystem,
		EditFn: func(invoice *billing.StandardInvoice) error {
			invoice.Description = lo.ToPtr("edited")

			return nil
		},
	})
	s.Require().NoError(err)

	// then: the line is stamped
	s.requireInvoicingAppTaxCode(invoiceID, lineID, lo.ToPtr("sandbox-code"))
}

// TestInvoicingAppTaxCodeRestampedInDraftAndFrozenAfter checks that a mapping added while the
// invoice is a draft reaches the line on the next draft calculation, and that a mapping added
// after the invoice left draft does not.
func (s *LineTaxConfigTestSuite) TestInvoicingAppTaxCodeRestampedInDraftAndFrozenAfter() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-app-code-freeze")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: a draft waiting for manual approval, with two lines whose Tax Codes only map Stripe
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
		profile.WorkflowConfig.Invoicing.AutoAdvance = false
		profile.WorkflowConfig.Invoicing.DraftPeriod = lo.Must(datetime.ISODurationString("PT0S").Parse())
	}))
	cust := s.CreateTestCustomer(namespace, "test")
	editedInDraft := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	editedAfterDraft := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_20000000")

	s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{
		"editedInDraft":    {TaxCodeID: editedInDraft.ID},
		"editedAfterDraft": {TaxCodeID: editedAfterDraft.ID},
	})

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoices[0].Status)

	invoiceID := invoices[0].GetInvoiceID()
	lines := linesByName(invoices[0])
	inDraftLineID := lines["editedInDraft"].ID
	afterDraftLineID := lines["editedAfterDraft"].ID

	s.requireInvoicingAppTaxCode(invoiceID, inDraftLineID, lo.ToPtr(""))
	s.requireInvoicingAppTaxCode(invoiceID, afterDraftLineID, lo.ToPtr(""))

	s.Run("when: a sandbox mapping is added and the draft is recalculated", func() {
		s.addSandboxMapping(editedInDraft, "sandbox-in-draft")

		_, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      invoiceID,
			ChangeSource: billing.ChangeSourceSystem,
			EditFn: func(invoice *billing.StandardInvoice) error {
				invoice.Description = lo.ToPtr("edited")

				return nil
			},
		})
		s.Require().NoError(err)
	})

	s.Run("then: the line picks the new mapping up, the other stays empty", func() {
		s.requireInvoicingAppTaxCode(invoiceID, inDraftLineID, lo.ToPtr("sandbox-in-draft"))
		s.requireInvoicingAppTaxCode(invoiceID, afterDraftLineID, lo.ToPtr(""))
	})

	s.Run("when: the invoice leaves draft and a sandbox mapping is added afterwards", func() {
		approved, err := s.BillingService.ApproveInvoice(ctx, invoiceID)
		s.Require().NoError(err)
		s.Require().Equal(billing.StandardInvoiceStatusPaid, approved.Status)

		s.addSandboxMapping(editedAfterDraft, "sandbox-after-draft")
	})

	s.Run("then: both stamps are frozen, in the database and through the billing service", func() {
		s.requireInvoicingAppTaxCode(invoiceID, inDraftLineID, lo.ToPtr("sandbox-in-draft"))
		s.requireInvoicingAppTaxCode(invoiceID, afterDraftLineID, lo.ToPtr(""))

		s.Equal("", s.readStandardLine(invoiceID, afterDraftLineID).BilledTaxCode(app.AppTypeSandbox))
	})
}

// addSandboxMapping adds a sandbox app mapping to a Tax Code; mappings can only be added.
func (s *LineTaxConfigTestSuite) addSandboxMapping(tc taxcode.TaxCode, code string) {
	s.T().Helper()

	_, err := s.TaxCodeEnv.Service.UpdateTaxCode(s.T().Context(), taxcode.UpdateTaxCodeInput{
		NamespacedID: tc.NamespacedID,
		Name:         tc.Name,
		Description:  tc.Description,
		AppMappings:  append(slices.Clone(tc.AppMappings), taxcode.TaxCodeAppMapping{AppType: app.AppTypeSandbox, TaxCode: code}),
		Metadata:     tc.Metadata,
		Annotations:  tc.Annotations,
	})
	s.Require().NoError(err)
}

// TestDeletedTaxCodeStillStamped checks that a line whose Tax Code is soft-deleted after the line
// was created, but before the invoice was calculated, is stamped with that code's mapping.
func (s *LineTaxConfigTestSuite) TestDeletedTaxCodeStillStamped() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-app-code-deleted")
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)

	// given: a pending line with a mapped Tax Code that is deleted before the line is collected
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	cust := s.CreateTestCustomer(namespace, "test")
	mapped := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeSandbox, "sandbox-code")

	s.createPendingLines(namespace, cust, now, map[string]*productcatalog.TaxCodeConfig{
		"mapped": {TaxCodeID: mapped.ID},
	})
	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: mapped.NamespacedID}))

	// when: the line is collected into a draft invoice
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)

	// then: the line carries the deleted Tax Code's mapping
	s.requireInvoicingAppTaxCode(invoices[0].GetInvoiceID(), linesByName(invoices[0])["mapped"].ID, lo.ToPtr("sandbox-code"))
}

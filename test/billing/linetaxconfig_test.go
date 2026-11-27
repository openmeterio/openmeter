package billing

import (
	"encoding/json"
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

// LineTaxConfigTestSuite covers the tax_config JSON snapshot persisted next to the
// tax_code_id/tax_behavior columns of invoice lines. The domain only carries the Tax Code ID
// and behavior; the JSON keeps the legacy shape (stripe.code, tax_code) for stored history.
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

// storedTaxConfigMap returns the persisted tax_config snapshot of a line as a generic map, for
// assertions on the embedded Tax Code entity that don't pin the entity's full shape.
func (s *LineTaxConfigTestSuite) storedTaxConfigMap(lineID string) map[string]any {
	s.T().Helper()

	var out map[string]any
	s.Require().NoError(json.Unmarshal([]byte(s.storedTaxConfigJSON(lineID)), &out))

	return out
}

// TestStoredTaxConfigOnWrite checks the legacy-shaped snapshot written on insert: gathering lines
// carry stripe.code but never the embedded Tax Code, standard lines carry both, and the snapshot
// only holds what the line actually has.
func (s *LineTaxConfigTestSuite) TestStoredTaxConfigOnWrite() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-stored-write")
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

	s.Run("then: gathering lines store the snapshot without the Tax Code", func() {
		gatheringByName := lo.KeyBy(pending.Lines, func(line billing.GatheringLine) string { return line.Name })

		mapped := gatheringByName["mapped"]
		s.Equal(mappedTaxCode.ID, lo.FromPtr(s.storedLine(mapped.ID).TaxCodeID))
		s.JSONEq(
			`{"behavior":"exclusive","tax_code_id":"`+mappedTaxCode.ID+`","stripe":{"code":"txcd_10000000"}}`,
			s.storedTaxConfigJSON(mapped.ID),
		)

		unmapped := gatheringByName["unmapped"]
		s.Equal(unmappedTaxCode.ID, lo.FromPtr(s.storedLine(unmapped.ID).TaxCodeID))
		s.JSONEq(`{"tax_code_id":"`+unmappedTaxCode.ID+`"}`, s.storedTaxConfigJSON(unmapped.ID))

		behaviorOnly := gatheringByName["behavior-only"]
		s.Nil(s.storedLine(behaviorOnly.ID).TaxCodeID)
		s.JSONEq(`{"behavior":"inclusive"}`, s.storedTaxConfigJSON(behaviorOnly.ID))
	})

	s.Run("then: standard lines store the snapshot with the Tax Code", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     &now,
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)

		standardByName := lo.KeyBy(invoices[0].Lines.OrEmpty(), func(line *billing.StandardLine) string { return line.Name })
		s.Require().Len(standardByName, 3)

		mapped := s.storedTaxConfigMap(standardByName["mapped"].ID)
		s.Equal("exclusive", mapped["behavior"])
		s.Equal(mappedTaxCode.ID, mapped["tax_code_id"])
		s.Equal(map[string]any{"code": "txcd_10000000"}, mapped["stripe"])
		s.Require().IsType(map[string]any{}, mapped["tax_code"])
		s.Equal(mappedTaxCode.ID, mapped["tax_code"].(map[string]any)["id"])

		unmapped := s.storedTaxConfigMap(standardByName["unmapped"].ID)
		s.Equal(unmappedTaxCode.ID, unmapped["tax_code_id"])
		s.NotContains(unmapped, "stripe", "a Tax Code without a Stripe mapping must not store a stripe code")
		s.Require().IsType(map[string]any{}, unmapped["tax_code"])
		s.Equal(unmappedTaxCode.ID, unmapped["tax_code"].(map[string]any)["id"])

		behaviorOnlyLine := standardByName["behavior-only"]
		s.Nil(s.storedLine(behaviorOnlyLine.ID).TaxCodeID)
		s.JSONEq(`{"behavior":"inclusive"}`, s.storedTaxConfigJSON(behaviorOnlyLine.ID))
	})
}

// TestStoredTaxConfigKeptOnLinePastDraftUpsert checks the history guard: once a line is
// re-upserted for a reason unrelated to tax (here an invoicing app merging line external IDs on
// the draft -> issuing transition), the stored tax_config snapshot of the original write stays as
// is, even if it no longer matches what the current Tax Code would render.
func (s *LineTaxConfigTestSuite) TestStoredTaxConfigKeptOnLinePastDraftUpsert() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-line-tax-history-guard")
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

	// and: the stored snapshot predates a remap of the Tax Code (older stripe code and Tax Code
	// state), while tax_code_id and tax_behavior stay consistent with it
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
	s.Require().NoError(s.DBClient.BillingInvoiceLine.UpdateOneID(lineID).SetTaxConfig(legacySnapshot).Exec(ctx))

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

	// and: the stored snapshot is untouched, the columns still hold the tax identity
	s.Equal(legacyJSON, s.storedTaxConfigJSON(lineID))

	row := s.storedLine(lineID)
	s.Equal(taxCode.ID, lo.FromPtr(row.TaxCodeID))
	s.Equal(productcatalog.ExclusiveTaxBehavior, lo.FromPtr(row.TaxBehavior))
}

// TestDraftLineWithDeletedTaxCodeCanBeResavedAndKeepsSnapshot checks that a draft line inheriting a
// soft-deleted Tax Code still passes calculation and upsert, and that its stored snapshot embeds
// the deleted Tax Code.
func (s *LineTaxConfigTestSuite) TestDraftLineWithDeletedTaxCodeCanBeResavedAndKeepsSnapshot() {
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

	s.Run("then: the snapshot embeds the deleted Tax Code", func() {
		s.requireDeletedTaxCodeSnapshot(invoice.Lines.OrEmpty()[0].ID, taxCode.ID)
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

		s.requireDeletedTaxCodeSnapshot(edited.Lines.OrEmpty()[0].ID, taxCode.ID)
	})
}

// requireDeletedTaxCodeSnapshot requires the line's stored tax_config to reference the Tax Code
// and embed it as deleted.
func (s *LineTaxConfigTestSuite) requireDeletedTaxCodeSnapshot(lineID string, taxCodeID string) {
	s.T().Helper()

	row := s.storedLine(lineID)
	s.Equal(taxCodeID, lo.FromPtr(row.TaxCodeID))
	s.Equal(taxCodeID, lo.FromPtr(row.TaxConfig.TaxCodeID))
	s.Require().NotNil(row.TaxConfig.TaxCode)
	s.Equal(taxCodeID, row.TaxConfig.TaxCode.ID)
	s.NotNil(row.TaxConfig.TaxCode.DeletedAt, "the embedded Tax Code is the deleted one")
}

package creditsdisabled

import (
	"encoding/json"
	"fmt"
	"maps"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// These tests pin the v1 taxConfig response shape on the billing paths that work differently
// when credits are disabled: invoice lines skip charges, so they keep the tax config they were
// given instead of getting the organization's default tax code. The catalog and billing profile
// paths are the same in both modes and are covered by e2e/taxconfig_v1_test.go.
//
// Don't edit the expectations to make a refactor pass: a diff here is a v1 contract change.

// v1TaxConfigScenario is one legacy taxConfig input a v1 client can send, with the taxConfig
// it comes back with when stored as given.
type v1TaxConfigScenario struct {
	key   string
	input *api.TaxConfig
	want  *api.TaxConfig
}

// v1TaxConfigScenarios covers the inputs v1 accepts. taxCode must have a Stripe app mapping:
// the response renders stripe.code from the tax code's mapping, not from the input.
func v1TaxConfigScenarios(taxCode v3sdk.TaxCode) []v1TaxConfigScenario {
	stripeCode := stripeAppMapping(taxCode)

	resolved := &api.TaxConfig{
		Stripe:    &api.StripeTaxConfig{Code: stripeCode},
		TaxCodeId: lo.ToPtr(taxCode.ID),
	}

	return []v1TaxConfigScenario{
		{
			key:   "stripe_code_only",
			input: &api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: stripeCode}},
			want:  resolved,
		},
		{
			key:   "tax_code_id_only",
			input: &api.TaxConfig{TaxCodeId: lo.ToPtr(taxCode.ID)},
			want:  resolved,
		},
		{
			// The ID wins, and the client's stripe.code is replaced by the tax code's mapping.
			key: "tax_code_id_wins_over_stripe_code",
			input: &api.TaxConfig{
				Stripe:    &api.StripeTaxConfig{Code: randomStripeTaxCode()},
				TaxCodeId: lo.ToPtr(taxCode.ID),
			},
			want: resolved,
		},
		{
			key:   "behavior_only",
			input: &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)},
			want:  &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)},
		},
		{
			key: "none",
		},
	}
}

func uniqueKey(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), rand.IntN(10_000))
}

// randomStripeTaxCode returns a Stripe code that is very unlikely to be mapped already, since
// all e2e tests share one namespace and a mapped Stripe code resolves to the existing tax code.
func randomStripeTaxCode() string {
	return fmt.Sprintf("txcd_%08d", rand.IntN(100_000_000))
}

// Tax codes only have a v3 API.
func newV3Client(t *testing.T) *v3sdk.Client {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	client, err := v3sdk.New(strings.TrimRight(address, "/") + "/api/v3")
	require.NoError(t, err)

	return client
}

func createStripeMappedTaxCode(t *testing.T, c *v3sdk.Client) v3sdk.TaxCode {
	t.Helper()

	taxCode, err := c.Tax.CreateCode(t.Context(), v3sdk.CreateTaxCodeRequest{
		Key:  uniqueKey("v1_taxconfig"),
		Name: "v1 taxConfig characterization",
		AppMappings: []v3sdk.TaxCodeAppMapping{{
			AppType: v3sdk.AppTypeStripe,
			TaxCode: randomStripeTaxCode(),
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, taxCode)

	return *taxCode
}

func stripeAppMapping(taxCode v3sdk.TaxCode) string {
	mapping, _ := lo.Find(taxCode.AppMappings, func(m v3sdk.TaxCodeAppMapping) bool {
		return m.AppType == v3sdk.AppTypeStripe
	})

	return mapping.TaxCode
}

// providerDefaultTaxCodeID returns the ID of the tax code the server provisions with the
// "default" key ("Provider default"), which invoice edits give lines without a tax code. Its ID
// differs per namespace, but its key is fixed.
func providerDefaultTaxCodeID(t *testing.T, c *v3sdk.Client) string {
	t.Helper()

	for taxCode, err := range c.Tax.ListCodesAll(t.Context(), v3sdk.TaxCodeListParams{}) {
		require.NoError(t, err)

		if taxCode.Key == "default" {
			return taxCode.ID
		}
	}

	require.Fail(t, "default tax code not found")

	return ""
}

// createV1TaxConfigFeatures creates one metered feature per scenario, keyed by scenario, so
// every scenario can also be expressed as a usage based line or rate card.
func createV1TaxConfigFeatures(t *testing.T, client *api.ClientWithResponses, scenarios []v1TaxConfigScenario) map[string]string {
	t.Helper()

	meterSlug := uniqueKey("v1_taxconfig_meter")
	meterResp, err := client.CreateMeterWithResponse(t.Context(), api.MeterCreate{
		Slug:          meterSlug,
		Aggregation:   api.MeterAggregationSum,
		EventType:     meterSlug,
		ValueProperty: lo.ToPtr("$.value"),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, meterResp.StatusCode(), "create meter: %s", string(meterResp.Body))

	featureKeys := make(map[string]string, len(scenarios))
	for _, scenario := range scenarios {
		featureResp, err := client.CreateFeatureWithResponse(t.Context(), api.CreateFeatureJSONRequestBody{
			Key:       uniqueKey("usage_" + scenario.key),
			Name:      "v1 taxConfig " + scenario.key,
			MeterSlug: lo.ToPtr(meterSlug),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, featureResp.StatusCode(), "create feature: %s", string(featureResp.Body))
		require.NotNil(t, featureResp.JSON201)

		featureKeys[scenario.key] = featureResp.JSON201.Key
	}

	return featureKeys
}

func createV1TaxConfigCustomer(t *testing.T, client *api.ClientWithResponses, keyPrefix string) api.Customer {
	t.Helper()

	resp, err := client.CreateCustomerWithResponse(t.Context(), api.CustomerCreate{
		Name:     "v1 taxConfig " + keyPrefix,
		Key:      lo.ToPtr(uniqueKey(keyPrefix)),
		Currency: lo.ToPtr(api.CurrencyCode("USD")),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode(), "create customer: %s", string(resp.Body))
	require.NotNil(t, resp.JSON201)

	return *resp.JSON201
}

// assertV1TaxConfigs checks that every expected key is present in got with the expected
// taxConfig. Extra keys in got are ignored.
func assertV1TaxConfigs(t *testing.T, want, got map[string]*api.TaxConfig) {
	t.Helper()

	for key, wantTaxConfig := range want {
		gotTaxConfig, ok := got[key]
		if assert.True(t, ok, "missing %s", key) {
			assert.Equal(t, wantTaxConfig, gotTaxConfig, "taxConfig of %s", key)
		}
	}
}

// v1InvoiceLineTaxConfigs returns the taxConfig of each line by name. It also checks that the
// line's rateCard.taxConfig and the taxConfig of its detailed lines match the line's own.
func v1InvoiceLineTaxConfigs(t *testing.T, lines []api.InvoiceLine) map[string]*api.TaxConfig {
	t.Helper()

	got := make(map[string]*api.TaxConfig, len(lines))
	for _, line := range lines {
		got[line.Name] = line.TaxConfig

		if assert.NotNil(t, line.RateCard, "rate card of line %s", line.Name) {
			assert.Equal(t, line.TaxConfig, line.RateCard.TaxConfig, "rateCard.taxConfig of line %s", line.Name)
		}

		for _, child := range lo.FromPtr(line.Children) {
			assert.Equal(t, line.TaxConfig, child.TaxConfig, "taxConfig of detailed line %s of %s", child.Name, line.Name)

			if assert.NotNil(t, child.RateCard, "rate card of detailed line %s of %s", child.Name, line.Name) {
				assert.Equal(t, line.TaxConfig, child.RateCard.TaxConfig, "rateCard.taxConfig of detailed line %s of %s", child.Name, line.Name)
			}
		}
	}

	return got
}

// createManualApprovalBillingProfile creates a profile from the default one without
// auto-advance, so standard invoices stay in draft and can be edited, and with an exclusive
// default tax behavior, so the invoice-level default is visible on the lines.
func createManualApprovalBillingProfile(t *testing.T, client *api.ClientWithResponses) api.BillingProfile {
	t.Helper()

	expand := []api.BillingProfileExpand{api.BillingProfileExpandApps}
	resp, err := client.ListBillingProfilesWithResponse(t.Context(), &api.ListBillingProfilesParams{
		Expand:   &expand,
		PageSize: lo.ToPtr(api.PaginationPageSize(100)),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "list billing profiles: %s", string(resp.Body))
	require.NotNil(t, resp.JSON200)

	defaultProfile, found := lo.Find(resp.JSON200.Items, func(p api.BillingProfile) bool {
		return p.Default
	})
	require.True(t, found, "default billing profile is required")

	var appRefs struct {
		Invoicing struct {
			ID string `json:"id"`
		} `json:"invoicing"`
		Payment struct {
			ID string `json:"id"`
		} `json:"payment"`
		Tax struct {
			ID string `json:"id"`
		} `json:"tax"`
	}
	rawApps, err := json.Marshal(defaultProfile.Apps)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rawApps, &appRefs), "billing profile apps: %s", string(rawApps))

	invoicing := api.BillingWorkflowInvoicingSettings{}
	if defaultProfile.Workflow.Invoicing != nil {
		invoicing = *defaultProfile.Workflow.Invoicing
	}

	invoicing.AutoAdvance = lo.ToPtr(false)
	invoicing.DefaultTaxConfig = &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}

	createResp, err := client.CreateBillingProfileWithResponse(t.Context(), api.BillingProfileCreate{
		Name: uniqueKey("v1_taxconfig_profile"),
		Apps: api.BillingProfileAppsCreate{
			Invoicing: appRefs.Invoicing.ID,
			Payment:   appRefs.Payment.ID,
			Tax:       appRefs.Tax.ID,
		},
		Supplier: defaultProfile.Supplier,
		Workflow: api.BillingWorkflowCreate{
			Collection: defaultProfile.Workflow.Collection,
			Invoicing:  &invoicing,
			Payment:    defaultProfile.Workflow.Payment,
			Tax:        defaultProfile.Workflow.Tax,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create billing profile: %s", string(createResp.Body))
	require.NotNil(t, createResp.JSON201)

	return *createResp.JSON201
}

// withProfileBehavior returns want as a standard invoice returns it: standard invoices merge
// the invoice's defaultTaxConfig into their lines, so lines without a behavior get the
// profile's exclusive one. Gathering invoices don't.
func withProfileBehavior(want map[string]*api.TaxConfig) map[string]*api.TaxConfig {
	merged := make(map[string]*api.TaxConfig, len(want))
	for key, taxConfig := range want {
		mergedTaxConfig := lo.FromPtr(taxConfig)
		if mergedTaxConfig.Behavior == nil {
			mergedTaxConfig.Behavior = lo.ToPtr(api.TaxBehaviorExclusive)
		}

		merged[key] = &mergedTaxConfig
	}

	return merged
}

func TestV1TaxConfigInvoiceLines(t *testing.T) {
	client := initClient(t)
	c := newV3Client(t)

	taxCode := createStripeMappedTaxCode(t, c)
	scenarios := v1TaxConfigScenarios(taxCode)
	featureKeys := createV1TaxConfigFeatures(t, client, scenarios)

	profile := createManualApprovalBillingProfile(t, client)
	customer := createV1TaxConfigCustomer(t, client, "v1_taxconfig_line_customer")

	overrideResp, err := client.UpsertBillingProfileCustomerOverrideWithResponse(t.Context(), customer.Id, api.BillingProfileCustomerOverrideCreate{
		BillingProfileId: lo.ToPtr(profile.Id),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, overrideResp.StatusCode(), "pin customer to billing profile: %s", string(overrideResp.Body))

	periodEnd := time.Now().UTC().Truncate(time.Second)
	period := api.Period{From: periodEnd.Add(-24 * time.Hour), To: periodEnd}

	flatPrice := func() *api.RateCardUsageBasedPrice {
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      "10",
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		return &price
	}

	unitPrice := func() *api.RateCardUsageBasedPrice {
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
			Amount: "0.1",
			Type:   api.UnitPriceWithCommitmentsTypeUnit,
		}))

		return &price
	}

	// v1 accepts a line's tax config in rateCard.taxConfig, or in the deprecated top-level
	// taxConfig next to a top-level price. lineInput holds either shape.
	type lineInput struct {
		name      string
		invoiceAt time.Time
		rateCard  *api.InvoiceUsageBasedRateCard
		price     *api.RateCardUsageBasedPrice
		taxConfig *api.TaxConfig
	}

	// lineInputs returns, per scenario, a flat fee line, a usage based line and a flat fee line
	// in the deprecated shape, named prefix+{flat,usage,legacy}_+scenario, keyed by the
	// scenario they express.
	lineInputs := func(prefix string) ([]lineInput, map[string]string) {
		inputs := make([]lineInput, 0, 3*len(scenarios))
		scenarioOf := make(map[string]string, 3*len(scenarios))

		for _, scenario := range scenarios {
			flat := lineInput{
				name:      prefix + "flat_" + scenario.key,
				invoiceAt: period.From,
				rateCard:  &api.InvoiceUsageBasedRateCard{Price: flatPrice(), TaxConfig: scenario.input},
			}
			// Usage is billed in arrears, at the end of the period.
			usage := lineInput{
				name:      prefix + "usage_" + scenario.key,
				invoiceAt: period.To,
				rateCard: &api.InvoiceUsageBasedRateCard{
					FeatureKey: lo.ToPtr(featureKeys[scenario.key]),
					Price:      unitPrice(),
					TaxConfig:  scenario.input,
				},
			}
			legacy := lineInput{
				name:      prefix + "legacy_" + scenario.key,
				invoiceAt: period.From,
				price:     flatPrice(),
				taxConfig: scenario.input,
			}

			for _, input := range []lineInput{flat, usage, legacy} {
				inputs = append(inputs, input)
				scenarioOf[input.name] = scenario.key
			}
		}

		return inputs, scenarioOf
	}

	// wantFor maps each line to the taxConfig its scenario comes back with in scenarioWant.
	wantFor := func(scenarioOf map[string]string, scenarioWant map[string]*api.TaxConfig) map[string]*api.TaxConfig {
		want := make(map[string]*api.TaxConfig, len(scenarioOf))
		for name, scenario := range scenarioOf {
			want[name] = scenarioWant[scenario]
		}

		return want
	}

	// Without charges, lines are stored with the tax config they were given: no default tax
	// code is filled in.
	storedAsGiven := make(map[string]*api.TaxConfig, len(scenarios))
	for _, scenario := range scenarios {
		storedAsGiven[scenario.key] = scenario.want
	}

	// An invoice edit gives every line without a tax code, including lines it only echoes
	// back, the tax code with the "default" key. See sanitizeTaxConfigForDiff in
	// openmeter/billing/service/invoiceupdate.go.
	defaultTaxCodeID := providerDefaultTaxCodeID(t, c)

	afterEdit := make(map[string]*api.TaxConfig, len(scenarios))
	for _, scenario := range scenarios {
		taxConfig := lo.FromPtr(scenario.want)
		if taxConfig.TaxCodeId == nil {
			taxConfig.TaxCodeId = lo.ToPtr(defaultTaxCodeID)
		}

		afterEdit[scenario.key] = &taxConfig
	}

	expand := []api.InvoiceExpand{api.InvoiceExpandLines}

	getInvoice := func(t *testing.T, invoiceID string) api.Invoice {
		t.Helper()

		resp, err := client.GetInvoiceWithResponse(t.Context(), invoiceID, &api.GetInvoiceParams{Expand: &expand})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "get invoice: %s", string(resp.Body))
		require.NotNil(t, resp.JSON200)
		require.NotNil(t, resp.JSON200.Lines)

		return *resp.JSON200
	}

	echoLines := func(t *testing.T, invoice api.Invoice) []api.InvoiceLineReplaceUpdate {
		t.Helper()

		return lo.Map(*invoice.Lines, func(line api.InvoiceLine, _ int) api.InvoiceLineReplaceUpdate {
			return api.InvoiceLineReplaceUpdate{
				Id:          lo.ToPtr(line.Id),
				Name:        line.Name,
				Description: line.Description,
				FeatureKey:  line.FeatureKey,
				InvoiceAt:   line.InvoiceAt,
				Metadata:    line.Metadata,
				Period:      line.Period,
				Price:       line.Price,
				RateCard:    line.RateCard,
				TaxConfig:   line.TaxConfig,
			}
		})
	}

	newLines := func(inputs []lineInput) []api.InvoiceLineReplaceUpdate {
		return lo.Map(inputs, func(in lineInput, _ int) api.InvoiceLineReplaceUpdate {
			return api.InvoiceLineReplaceUpdate{
				Name:      in.name,
				InvoiceAt: in.invoiceAt,
				Period:    period,
				RateCard:  in.rateCard,
				Price:     in.price,
				TaxConfig: in.taxConfig,
			}
		})
	}

	// updateInvoice replaces the invoice's lines and keeps everything else, except the edited
	// workflow defaultTaxConfig when one is given.
	updateInvoice := func(t *testing.T, invoice api.Invoice, lines []api.InvoiceLineReplaceUpdate, defaultTaxConfig *api.TaxConfig) api.Invoice {
		t.Helper()

		invoicing := api.InvoiceWorkflowInvoicingSettingsReplaceUpdate{}
		if settings := invoice.Workflow.Workflow.Invoicing; settings != nil {
			invoicing = api.InvoiceWorkflowInvoicingSettingsReplaceUpdate{
				AutoAdvance:                  settings.AutoAdvance,
				DefaultTaxConfig:             settings.DefaultTaxConfig,
				DraftPeriod:                  settings.DraftPeriod,
				DueAfter:                     settings.DueAfter,
				SubscriptionEndProrationMode: settings.SubscriptionEndProrationMode,
			}
		}

		if defaultTaxConfig != nil {
			invoicing.DefaultTaxConfig = defaultTaxConfig
		}

		resp, err := client.UpdateInvoiceWithResponse(t.Context(), invoice.Id, api.InvoiceReplaceUpdate{
			Customer: api.BillingPartyReplaceUpdate{
				Addresses: invoice.Customer.Addresses,
				Key:       invoice.Customer.Key,
				Name:      invoice.Customer.Name,
				TaxId:     invoice.Customer.TaxId,
			},
			Supplier: api.BillingPartyReplaceUpdate{
				Addresses: invoice.Supplier.Addresses,
				Key:       invoice.Supplier.Key,
				Name:      invoice.Supplier.Name,
				TaxId:     invoice.Supplier.TaxId,
			},
			Description: invoice.Description,
			Metadata:    invoice.Metadata,
			Lines:       lines,
			Workflow: api.InvoiceWorkflowReplaceUpdate{
				Workflow: api.InvoiceWorkflowSettingsReplaceUpdate{
					Invoicing: invoicing,
					Payment:   lo.FromPtr(invoice.Workflow.Workflow.Payment),
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "update invoice: %s", string(resp.Body))
		require.NotNil(t, resp.JSON200)
		require.NotNil(t, resp.JSON200.Lines)

		return *resp.JSON200
	}

	createdInputs, createdScenarioOf := lineInputs("")
	createdWant := wantFor(createdScenarioOf, storedAsGiven)

	createResp, err := client.CreatePendingInvoiceLineWithResponse(t.Context(), customer.Id, api.InvoicePendingLineCreateInput{
		Currency: "USD",
		Lines: lo.Map(createdInputs, func(in lineInput, _ int) api.InvoicePendingLineCreate {
			return api.InvoicePendingLineCreate{
				Name:      in.name,
				InvoiceAt: in.invoiceAt,
				Period:    period,
				RateCard:  in.rateCard,
				Price:     in.price,
				TaxConfig: in.taxConfig,
			}
		}),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create pending lines: %s", string(createResp.Body))
	require.NotNil(t, createResp.JSON201)

	gatheringInvoiceID := createResp.JSON201.Invoice.Id
	t.Logf("customer=%s gathering invoice=%s", customer.Id, gatheringInvoiceID)

	t.Run("create response", func(t *testing.T) {
		assertV1TaxConfigs(t, createdWant, v1InvoiceLineTaxConfigs(t, createResp.JSON201.Lines))
	})

	t.Run("gathering invoice", func(t *testing.T) {
		assertV1TaxConfigs(t, createdWant, v1InvoiceLineTaxConfigs(t, *getInvoice(t, gatheringInvoiceID).Lines))
	})

	gatheringEditedInputs, gatheringEditedScenarioOf := lineInputs("gathering_edited_")

	// Both the echoed lines and the added ones come back with the default tax code.
	createdAfterEditWant := wantFor(createdScenarioOf, afterEdit)
	gatheringEditedWant := wantFor(gatheringEditedScenarioOf, afterEdit)

	// Edits run at the top level, so every subtest only reads and can run on its own.
	gatheringInvoice := getInvoice(t, gatheringInvoiceID)
	gatheringEdited := updateInvoice(t, gatheringInvoice, append(echoLines(t, gatheringInvoice), newLines(gatheringEditedInputs)...), nil)

	t.Run("gathering invoice edit adds lines and fills in the default tax code", func(t *testing.T) {
		// given: the gathering invoice with the created lines, stored as given
		// when: the edit above echoes those lines and adds a line per scenario and shape
		// then: every line without a tax code, echoed or added, gets the default tax code
		got := v1InvoiceLineTaxConfigs(t, *gatheringEdited.Lines)
		assertV1TaxConfigs(t, createdAfterEditWant, got)
		assertV1TaxConfigs(t, gatheringEditedWant, got)

		got = v1InvoiceLineTaxConfigs(t, *getInvoice(t, gatheringInvoiceID).Lines)
		assertV1TaxConfigs(t, createdAfterEditWant, got)
		assertV1TaxConfigs(t, gatheringEditedWant, got)
	})

	// Lines created after the edit never went through it, so they show what invoicing alone
	// does to a line without a tax code.
	lateInputs, lateScenarioOf := lineInputs("late_")
	lateWant := wantFor(lateScenarioOf, storedAsGiven)

	lateResp, err := client.CreatePendingInvoiceLineWithResponse(t.Context(), customer.Id, api.InvoicePendingLineCreateInput{
		Currency: "USD",
		Lines: lo.Map(lateInputs, func(in lineInput, _ int) api.InvoicePendingLineCreate {
			return api.InvoicePendingLineCreate{
				Name:      in.name,
				InvoiceAt: in.invoiceAt,
				Period:    period,
				RateCard:  in.rateCard,
				Price:     in.price,
				TaxConfig: in.taxConfig,
			}
		}),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, lateResp.StatusCode(), "create pending lines: %s", string(lateResp.Body))
	require.NotNil(t, lateResp.JSON201)

	t.Run("lines created after the edit", func(t *testing.T) {
		assertV1TaxConfigs(t, lateWant, v1InvoiceLineTaxConfigs(t, lateResp.JSON201.Lines))
	})

	invoiceResp, err := client.InvoicePendingLinesActionWithResponse(t.Context(), api.InvoicePendingLinesActionInput{
		CustomerId: customer.Id,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, invoiceResp.StatusCode(), "invoice pending lines: %s", string(invoiceResp.Body))
	require.NotNil(t, invoiceResp.JSON201)
	require.Len(t, *invoiceResp.JSON201, 1)

	invoice := getInvoice(t, (*invoiceResp.JSON201)[0].Id)
	t.Logf("invoice=%s status=%s", invoice.Id, invoice.StatusDetails.ExtendedStatus)

	standardWant := withProfileBehavior(createdAfterEditWant)
	maps.Copy(standardWant, withProfileBehavior(gatheringEditedWant))
	maps.Copy(standardWant, withProfileBehavior(lateWant))

	t.Run("standard invoice", func(t *testing.T) {
		for _, line := range *invoice.Lines {
			// Usage based lines without usage have no detailed lines.
			if !strings.Contains(line.Name, "usage_") {
				assert.NotEmpty(t, lo.FromPtr(line.Children), "detailed lines of %s", line.Name)
			}
		}

		assertV1TaxConfigs(t, standardWant, v1InvoiceLineTaxConfigs(t, *invoice.Lines))

		require.NotNil(t, invoice.Workflow.Workflow.Invoicing)
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, invoice.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	})

	// Unlike in credits mode, these lines aren't managed by charges, so an edit can change an
	// existing line's tax config. The edit also sends workflow.invoicing.defaultTaxConfig,
	// which the server accepts but doesn't apply.
	changed := &api.TaxConfig{TaxCodeId: lo.ToPtr(taxCode.ID), Behavior: lo.ToPtr(api.TaxBehaviorInclusive)}

	editLines := echoLines(t, invoice)
	for i := range editLines {
		if editLines[i].Name != "flat_none" {
			continue
		}

		rateCard := *editLines[i].RateCard
		rateCard.TaxConfig = changed

		editLines[i].TaxConfig = changed
		editLines[i].RateCard = &rateCard
	}

	editedInputs, editedScenarioOf := lineInputs("edited_")
	editLines = append(editLines, newLines(editedInputs)...)

	// The edit also fills in the default tax code on the late lines it echoes back.
	editedWant := maps.Clone(standardWant)
	maps.Copy(editedWant, withProfileBehavior(wantFor(lateScenarioOf, afterEdit)))
	maps.Copy(editedWant, withProfileBehavior(wantFor(editedScenarioOf, afterEdit)))
	editedWant["flat_none"] = &api.TaxConfig{
		Behavior:  lo.ToPtr(api.TaxBehaviorInclusive),
		Stripe:    &api.StripeTaxConfig{Code: stripeAppMapping(taxCode)},
		TaxCodeId: lo.ToPtr(taxCode.ID),
	}

	updated := updateInvoice(t, invoice, editLines, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorInclusive)})

	t.Run("standard invoice edit adds lines, changes a line's tax config and ignores the default tax config", func(t *testing.T) {
		// given: the standard invoice, with an exclusive default behavior
		// when: the edit above echoes its lines, changes flat_none's tax config, adds a line per
		// scenario and shape, and sets an inclusive default behavior
		// then: flat_none gets the new tax config, lines without a tax code get the default one,
		// and the invoice default stays exclusive
		assertV1TaxConfigs(t, editedWant, v1InvoiceLineTaxConfigs(t, *updated.Lines))

		require.NotNil(t, updated.Workflow.Workflow.Invoicing)
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, updated.Workflow.Workflow.Invoicing.DefaultTaxConfig)

		reread := getInvoice(t, invoice.Id)

		assertV1TaxConfigs(t, editedWant, v1InvoiceLineTaxConfigs(t, *reread.Lines))

		require.NotNil(t, reread.Workflow.Workflow.Invoicing)
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, reread.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	})
}

func TestV1TaxConfigSubscriptionInvoiceLines(t *testing.T) {
	client := initClient(t)
	c := newV3Client(t)

	scenarios := v1TaxConfigScenarios(createStripeMappedTaxCode(t, c))
	featureKeys := createV1TaxConfigFeatures(t, client, scenarios)

	// Gathering lines are named after their rate card: the flat fee key, or the feature key
	// for usage based ones.
	rateCards := make([]api.RateCard, 0, 2*len(scenarios))
	want := make(map[string]*api.TaxConfig, 2*len(scenarios))

	for _, scenario := range scenarios {
		flatFee := api.RateCard{}
		require.NoError(t, flatFee.FromRateCardFlatFee(api.RateCardFlatFee{
			Key:  scenario.key,
			Name: scenario.key,
			Type: api.RateCardFlatFeeTypeFlatFee,
			Price: &api.FlatPriceWithPaymentTerm{
				Amount:      "10",
				PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
				Type:        api.FlatPriceWithPaymentTermTypeFlat,
			},
			BillingCadence: lo.ToPtr("P1M"),
			TaxConfig:      scenario.input,
		}))

		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
			Amount: "0.1",
			Type:   api.UnitPriceWithCommitmentsTypeUnit,
		}))

		featureKey := featureKeys[scenario.key]

		usageBased := api.RateCard{}
		require.NoError(t, usageBased.FromRateCardUsageBased(api.RateCardUsageBased{
			Key:            featureKey,
			Name:           featureKey,
			Type:           api.RateCardUsageBasedTypeUsageBased,
			FeatureKey:     lo.ToPtr(featureKey),
			Price:          &price,
			BillingCadence: "P1M",
			TaxConfig:      scenario.input,
		}))

		rateCards = append(rateCards, flatFee, usageBased)

		// Without charges, subscription lines copy the item's tax config as is.
		want[scenario.key] = scenario.want
		want[featureKey] = scenario.want
	}

	planResp, err := client.CreatePlanWithResponse(t.Context(), api.PlanCreate{
		Key:            uniqueKey("v1_taxconfig_sub_plan"),
		Name:           "v1 taxConfig subscription plan",
		Currency:       "USD",
		BillingCadence: "P1M",
		Phases: []api.PlanPhase{{
			Key:       "default",
			Name:      "Default",
			RateCards: rateCards,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, planResp.StatusCode(), "create plan: %s", string(planResp.Body))
	require.NotNil(t, planResp.JSON201)

	publishResp, err := client.PublishPlanWithResponse(t.Context(), planResp.JSON201.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, publishResp.StatusCode(), "publish plan: %s", string(publishResp.Body))
	require.NotNil(t, publishResp.JSON200)

	customer := createV1TaxConfigCustomer(t, client, "v1_taxconfig_sub_customer")

	timing := &api.SubscriptionTiming{}
	require.NoError(t, timing.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

	subscriptionCreate := api.SubscriptionCreate{}
	require.NoError(t, subscriptionCreate.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
		Timing:     timing,
		CustomerId: lo.ToPtr(customer.Id),
		Plan: api.PlanReferenceInput{
			Key:     publishResp.JSON200.Key,
			Version: lo.ToPtr(publishResp.JSON200.Version),
		},
	}))

	subscriptionResp, err := client.CreateSubscriptionWithResponse(t.Context(), subscriptionCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, subscriptionResp.StatusCode(), "create subscription: %s", string(subscriptionResp.Body))
	require.NotNil(t, subscriptionResp.JSON201)
	t.Logf("customer=%s subscription=%s", customer.Id, subscriptionResp.JSON201.Id)

	customers := api.InvoiceListParamsCustomers{customer.Id}
	expand := api.InvoiceListParamsExpand{api.InvoiceExpandLines}

	var got map[string]*api.TaxConfig
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		resp, err := client.ListInvoicesWithResponse(t.Context(), &api.ListInvoicesParams{
			Customers: &customers,
			Expand:    &expand,
			PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
		})
		require.NoError(collect, err)
		require.Equal(collect, http.StatusOK, resp.StatusCode(), "list invoices: %s", string(resp.Body))
		require.NotNil(collect, resp.JSON200)

		got = map[string]*api.TaxConfig{}
		for _, invoice := range resp.JSON200.Items {
			for _, line := range lo.FromPtr(invoice.Lines) {
				if line.Subscription != nil {
					got[line.Name] = line.TaxConfig
				}
			}
		}

		for key := range want {
			require.Contains(collect, got, key)
		}
	}, time.Minute, time.Second)

	assertV1TaxConfigs(t, want, got)
}

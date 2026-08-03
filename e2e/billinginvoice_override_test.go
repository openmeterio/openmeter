package e2e

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestInvoiceEditFlatFeeManualOverrides(t *testing.T) {
	ctx := t.Context()
	c := newV3Client(t)
	v1 := initClient(t)

	prefix := invoiceOverrideTestKey("invoice_override")
	customerKey := prefix + "_customer"
	flatFeeToDeleteName := prefix + "_flatfee_delete"
	flatFeeToUpdateName := prefix + "_flatfee_update"
	flatFeeToKeepName := prefix + "_flatfee_keep"
	usageBasedName := prefix + "_usage"
	createdFlatFeeName := prefix + "_flatfee_create"
	createdFlatFeeAmount := "11"
	updatedFlatFeeAmount := "6"

	var customer *v3sdk.Customer
	var plan *v3sdk.Plan
	var subscription *v3sdk.BillingSubscription
	var invoice api.Invoice
	var deleteLine api.InvoiceLine
	var updateLine api.InvoiceLine
	var keepLine api.InvoiceLine
	var deleteLineSystemAmount string
	var updateLineSystemAmount string
	var editedInvoice api.Invoice

	runRequired(t, "creates invoice edit plan fixtures", func(t *testing.T) {
		// given:
		// - unique feature and meter keys for this test run
		// when:
		// - a plan is created with three in-arrears flat fees and one usage-based item
		// then:
		// - the plan is published and ready for subscription creation
		meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:         prefix + "_meter",
			Name:        "Invoice Override Meter " + prefix,
			Aggregation: v3sdk.MeterAggregationCount,
			EventType:   prefix + "_event",
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, meter)

		meteredFeature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:   prefix + "_metered_feature",
			Name:  "Invoice Override Metered Feature " + prefix,
			Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, meteredFeature)

		phaseKey := "phase_1"
		createdPlan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:              prefix + "_plan",
			Name:             "Invoice Override Plan " + prefix,
			Currency:         "USD",
			BillingCadence:   "P1M",
			ProRatingEnabled: lo.ToPtr(false),
			Phases: []v3sdk.PlanPhaseInput{{
				Key:  phaseKey,
				Name: "Invoice Override Phase",
				RateCards: []v3sdk.RateCardInput{
					manualOverrideFlatFeeRateCard(flatFeeToDeleteName, "2"),
					manualOverrideFlatFeeRateCard(flatFeeToUpdateName, "5"),
					manualOverrideFlatFeeRateCard(flatFeeToKeepName, "7"),
					manualOverrideUnitRateCard(meteredFeature.Key, usageBasedName, *meteredFeature, "1"),
				},
			}},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, createdPlan)

		plan, err = c.Plans.Publish(t.Context(), createdPlan.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)
		require.Equal(t, v3sdk.PlanStatusActive, plan.Status)
	})

	runRequired(t, "creates funded customer and starts subscription", func(t *testing.T) {
		// given:
		// - a customer pinned to a manual-approval billing profile
		// - promotional credits available for invoice allocation
		// - a published plan with flat-fee and usage-based items
		// when:
		// - the customer subscribes in credit-then-invoice mode
		// then:
		// - all subscription-created charges are controlled by OpenMeter
		require.NotNil(t, plan)

		createdCustomer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      customerKey,
			Name:     "Invoice Override Customer " + prefix,
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, createdCustomer)
		customer = createdCustomer

		profile := createNewBillingProfileFromDefault(t, c, prefix, func(profile *v3sdk.CreateBillingProfileRequest) {
			profile.Name = "Invoice Override Manual Approval " + prefix
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &v3sdk.WorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)

			sendInvoice := lo.Must(v3sdk.WorkflowPaymentSettingsFromWorkflowPaymentSendInvoiceSettings(v3sdk.WorkflowPaymentSendInvoiceSettings{
				CollectionMethod: v3sdk.CollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})
		_, err = c.Customers.Billing.Update(t.Context(), customer.ID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: &v3sdk.ProfileReference{ID: profile.ID},
		})
		c.requireStatus(http.StatusOK, err)

		_, err = c.Customers.Credits.Grants.Create(t.Context(), customer.ID, v3sdk.CreateCreditGrantRequest{
			Name:          "Invoice Override Promotional Credits " + prefix,
			Amount:        "100",
			Currency:      v3sdk.BillingCurrencyCode("USD"),
			FundingMethod: v3sdk.CreditFundingMethodNone,
		})
		c.requireStatus(http.StatusCreated, err)

		createdSubscription, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer:       v3sdk.SubscriptionChangeCustomer{ID: lo.ToPtr(customer.ID)},
			Plan:           v3sdk.SubscriptionChangePlan{ID: lo.ToPtr(plan.ID)},
			SettlementMode: lo.ToPtr(v3sdk.SettlementModeCreditThenInvoice),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, createdSubscription)
		subscription = createdSubscription

		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			charges := listChargesByName(collect, c, customer.ID, []v3sdk.ChargeStatus{
				v3sdk.ChargeStatusCreated,
				v3sdk.ChargeStatusActive,
				v3sdk.ChargeStatusFinal,
			})
			assertFlatFeeChargeLifecycleController(collect, charges, flatFeeToDeleteName, v3sdk.LifecycleControllerSystem)
			assertFlatFeeChargeLifecycleController(collect, charges, flatFeeToUpdateName, v3sdk.LifecycleControllerSystem)
			assertFlatFeeChargeLifecycleController(collect, charges, flatFeeToKeepName, v3sdk.LifecycleControllerSystem)
			assertUsageBasedChargeController(collect, charges, usageBasedName, v3sdk.LifecycleControllerSystem)
		}, time.Minute, time.Second)
	})

	runRequired(t, "cancels the subscription and collects a manually approved invoice", func(t *testing.T) {
		// given:
		// - subscription-created credit-then-invoice charges exist
		// when:
		// - the subscription is canceled and billing collects a standard invoice
		// then:
		// - the invoice is available for manual approval with editable charge-backed lines
		time.Sleep(2 * time.Second)

		canceledSubscription, err := c.Subscriptions.Cancel(t.Context(), subscription.ID, v3sdk.SubscriptionCancel{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, canceledSubscription)
		subscription = canceledSubscription

		invoice = waitForManualApprovalInvoice(t, v1, customer.ID)
		require.NotNil(t, invoice.Lines)
		require.NotEmpty(t, *invoice.Lines)

		linesBeforeEdit := *invoice.Lines
		require.Len(t, linesBeforeEdit, 4)
		deleteLine = requireInvoiceLineByName(t, linesBeforeEdit, flatFeeToDeleteName)
		updateLine = requireInvoiceLineByName(t, linesBeforeEdit, flatFeeToUpdateName)
		keepLine = requireInvoiceLineByName(t, linesBeforeEdit, flatFeeToKeepName)
		deleteLineSystemAmount = flatInvoiceLineAmount(t, deleteLine)
		updateLineSystemAmount = flatInvoiceLineAmount(t, updateLine)
		_ = requireInvoiceLineByName(t, linesBeforeEdit, usageBasedName)
	})

	runRequired(t, "edits flat-fee invoice lines through the invoice API", func(t *testing.T) {
		// given:
		// - a mutable standard invoice contains charge-backed flat-fee and usage-based lines
		// when:
		// - the API deletes one flat-fee line, increases one flat-fee price, and creates a new flat-fee line
		// then:
		// - the invoice edit succeeds and billing returns the edited invoice state
		replacementLines := make([]api.InvoiceLineReplaceUpdate, 0, len(*invoice.Lines))
		for _, line := range *invoice.Lines {
			if line.Id == deleteLine.Id {
				continue
			}

			lineUpdate := invoiceLineReplaceUpdateFromLine(t, line)
			if line.Id == updateLine.Id {
				price := flatInvoiceLinePrice(t, updatedFlatFeeAmount)
				require.NotNil(t, lineUpdate.RateCard, "line[%s] rate card", line.Id)
				lineUpdate.Price = &price
				lineUpdate.RateCard.Price = &price
			}

			replacementLines = append(replacementLines, lineUpdate)
		}
		replacementLines = append(replacementLines, api.InvoiceLineReplaceUpdate{
			Name:        createdFlatFeeName,
			Description: lo.ToPtr("Invoice Override Created Flat Fee " + prefix),
			InvoiceAt:   keepLine.InvoiceAt,
			Period:      keepLine.Period,
			RateCard:    flatInvoiceLineRateCard(t, createdFlatFeeAmount, keepLine.RateCard, nil),
		})

		updateResp, err := v1.UpdateInvoiceWithResponse(ctx, invoice.Id, api.InvoiceReplaceUpdate{
			Customer: billingPartyReplaceUpdateFromInvoiceCustomer(invoice.Customer),
			Supplier: billingPartyReplaceUpdateFromParty(invoice.Supplier),
			Workflow: invoiceWorkflowReplaceUpdateFromInvoice(invoice),
			Lines:    replacementLines,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, updateResp.StatusCode(), "update invoice: %s", string(updateResp.Body))
		require.NotNil(t, updateResp.JSON200)

		editedInvoice = getInvoiceWithDeletedLines(t, v1, invoice.Id)
		require.NotNil(t, editedInvoice.Lines)
	})

	t.Run("asserts manual override ownership and credit allocation after invoice edit", func(t *testing.T) {
		// given:
		// - an invoice edit has been persisted against charge-backed flat-fee lines
		// when:
		// - the edited invoice, customer credit balance, and customer charges are read back
		// then:
		// - edited flat-fee lines are manually managed
		// - untouched subscription-owned lines remain system managed
		// - promotional credits are allocated consistently
		linesAfterEdit := *editedInvoice.Lines
		require.Len(t, linesAfterEdit, 5)

		deletedLineAfterEdit := requireInvoiceLineByName(t, linesAfterEdit, flatFeeToDeleteName)
		require.NotNil(t, deletedLineAfterEdit.DeletedAt, "deleted flat-fee line")
		assert.Equal(t, api.InvoiceLineManagedByManual, deletedLineAfterEdit.ManagedBy)
		assert.Equal(t, deleteLine.Period, deletedLineAfterEdit.Period)
		assert.Equal(t, deleteLine.InvoiceAt, deletedLineAfterEdit.InvoiceAt)
		assert.Equal(t, "2", flatInvoiceLineAmount(t, deletedLineAfterEdit))
		requireInvoiceTotals(t, expectedInvoiceTotals{
			Amount:       2,
			CreditsTotal: 2,
		}, deletedLineAfterEdit.Totals)
		require.Equal(t, float64(2), invoiceLineCreditsApplied(t, deletedLineAfterEdit), "deleted flat-fee line credits")

		updatedLineAfterEdit := requireInvoiceLineByName(t, linesAfterEdit, flatFeeToUpdateName)
		assert.Equal(t, api.InvoiceLineManagedByManual, updatedLineAfterEdit.ManagedBy)
		assert.Nil(t, updatedLineAfterEdit.DeletedAt)
		assert.Equal(t, updateLine.Period, updatedLineAfterEdit.Period)
		assert.Equal(t, updateLine.InvoiceAt, updatedLineAfterEdit.InvoiceAt)
		assert.Equal(t, updateLine.Description, updatedLineAfterEdit.Description)
		assert.Equal(t, updateLine.FeatureKey, updatedLineAfterEdit.FeatureKey)
		assert.Equal(t, updatedFlatFeeAmount, flatInvoiceLineAmount(t, updatedLineAfterEdit))
		requireInvoiceTotals(t, expectedInvoiceTotals{
			Amount:       6,
			CreditsTotal: 6,
		}, updatedLineAfterEdit.Totals)
		require.Equal(t, float64(6), invoiceLineCreditsApplied(t, updatedLineAfterEdit), "updated flat-fee line credits")

		createdLineAfterEdit := requireInvoiceLineByName(t, linesAfterEdit, createdFlatFeeName)
		assert.Equal(t, api.InvoiceLineManagedByManual, createdLineAfterEdit.ManagedBy)
		assert.NotEmpty(t, createdLineAfterEdit.Id)
		assert.Nil(t, createdLineAfterEdit.DeletedAt)
		assert.Equal(t, lo.ToPtr("Invoice Override Created Flat Fee "+prefix), createdLineAfterEdit.Description)
		assert.Equal(t, keepLine.Period, createdLineAfterEdit.Period)
		assert.Equal(t, keepLine.InvoiceAt, createdLineAfterEdit.InvoiceAt)
		assert.Nil(t, createdLineAfterEdit.FeatureKey)
		assert.Equal(t, createdFlatFeeAmount, flatInvoiceLineAmount(t, createdLineAfterEdit))
		requireInvoiceTotals(t, expectedInvoiceTotals{
			Amount:       11,
			CreditsTotal: 11,
		}, createdLineAfterEdit.Totals)
		require.Equal(t, float64(11), invoiceLineCreditsApplied(t, createdLineAfterEdit), "created flat-fee line credits")

		keepLineAfterEdit := requireInvoiceLineByName(t, linesAfterEdit, flatFeeToKeepName)
		assert.Equal(t, api.InvoiceLineManagedBySubscription, keepLineAfterEdit.ManagedBy)
		assert.Nil(t, keepLineAfterEdit.DeletedAt)
		assert.Equal(t, keepLine.Period, keepLineAfterEdit.Period)
		assert.Equal(t, keepLine.InvoiceAt, keepLineAfterEdit.InvoiceAt)
		assert.Equal(t, keepLine.Description, keepLineAfterEdit.Description)
		assert.Equal(t, keepLine.FeatureKey, keepLineAfterEdit.FeatureKey)
		assert.Equal(t, "7", flatInvoiceLineAmount(t, keepLineAfterEdit))
		requireInvoiceTotals(t, expectedInvoiceTotals{
			Amount:       7,
			CreditsTotal: 7,
		}, keepLineAfterEdit.Totals)
		require.Equal(t, float64(7), invoiceLineCreditsApplied(t, keepLineAfterEdit), "kept flat-fee line credits")

		usageLineAfterEdit := requireInvoiceLineByName(t, linesAfterEdit, usageBasedName)
		assert.Equal(t, api.InvoiceLineManagedBySubscription, usageLineAfterEdit.ManagedBy)
		assert.Nil(t, usageLineAfterEdit.DeletedAt)
		requireInvoiceTotals(t, expectedInvoiceTotals{}, usageLineAfterEdit.Totals)
		require.Equal(t, float64(0), invoiceLineCreditsApplied(t, usageLineAfterEdit), "usage-based line credits")

		requireInvoiceTotals(t, expectedInvoiceTotals{
			Amount:       24,
			CreditsTotal: 24,
		}, editedInvoice.Totals)
		require.Equal(t, float64(24), activeInvoiceLinesCreditsApplied(t, linesAfterEdit), "active line-level promotional credits")
		require.Equal(t, float64(26), invoiceLinesCreditsApplied(t, linesAfterEdit), "all line-level promotional credits including deleted tombstones")

		balance, err := c.Customers.Credits.Balance.Get(t.Context(), customer.ID, v3sdk.GetCustomerCreditBalanceParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, balance)
		requireCustomerCreditBalance(t, balance, "USD", 76, 0)

		activeCharges := listChargesByName(t, c, customer.ID, []v3sdk.ChargeStatus{
			v3sdk.ChargeStatusCreated,
			v3sdk.ChargeStatusActive,
			v3sdk.ChargeStatusFinal,
		})
		requireChargeNames(t, activeCharges, flatFeeToUpdateName, flatFeeToKeepName, createdFlatFeeName, usageBasedName)
		assertFlatFeeChargeLifecycleController(t, activeCharges, flatFeeToUpdateName, v3sdk.LifecycleControllerManual)
		requireFlatFeeChargeIntentMatchesLine(t, activeCharges, updatedLineAfterEdit)
		requireFlatFeeChargeSystemIntentMatchesLine(t, activeCharges, flatFeeToUpdateName, updateLine, updateLineSystemAmount)
		assertFlatFeeChargeLifecycleController(t, activeCharges, flatFeeToKeepName, v3sdk.LifecycleControllerSystem)
		requireFlatFeeChargeIntentMatchesLine(t, activeCharges, keepLineAfterEdit)
		requireFlatFeeChargeHasNoSystemIntent(t, activeCharges, flatFeeToKeepName)
		assertFlatFeeChargeLifecycleController(t, activeCharges, createdFlatFeeName, v3sdk.LifecycleControllerManual)
		requireFlatFeeChargeIntentMatchesLine(t, activeCharges, createdLineAfterEdit)
		requireFlatFeeChargeHasNoSystemIntent(t, activeCharges, createdFlatFeeName)
		assertUsageBasedChargeController(t, activeCharges, usageBasedName, v3sdk.LifecycleControllerSystem)
		requireUsageBasedChargeHasNoSystemIntent(t, activeCharges, usageBasedName)

		deletedCharges := listChargesByName(t, c, customer.ID, []v3sdk.ChargeStatus{
			v3sdk.ChargeStatusDeleted,
		})
		requireChargeNames(t, deletedCharges, flatFeeToDeleteName)
		assertFlatFeeChargeLifecycleController(t, deletedCharges, flatFeeToDeleteName, v3sdk.LifecycleControllerManual)
		requireFlatFeeChargeIntentMatchesLine(t, deletedCharges, deletedLineAfterEdit)
		requireFlatFeeChargeSystemIntentMatchesLine(t, deletedCharges, flatFeeToDeleteName, deleteLine, deleteLineSystemAmount)
	})
}

func runRequired(t *testing.T, name string, fn func(t *testing.T)) {
	t.Helper()

	if !t.Run(name, fn) {
		t.FailNow()
	}
}

// createNewBillingProfileFromDefault clones the default billing profile and lets
// the caller edit the create request before it is persisted. The clone keeps
// supplier and app wiring from the environment so scenarios can change only the
// workflow controls relevant to the behavior under test.
func createNewBillingProfileFromDefault(t *testing.T, c *v3Client, prefix string, edit func(*v3sdk.CreateBillingProfileRequest)) v3sdk.Profile {
	t.Helper()

	profiles, err := c.Billing.ListProfiles(t.Context(), v3sdk.ProfileListParams{
		Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
	})
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, profiles)

	defaultIdx := slices.IndexFunc(profiles.Data, func(profile v3sdk.Profile) bool {
		return profile.Default
	})
	require.NotEqual(t, -1, defaultIdx, "default billing profile is required")

	base := profiles.Data[defaultIdx]
	supplier := base.Supplier
	if supplier.ID != nil && *supplier.ID == "" {
		supplier.ID = nil
	}

	request := v3sdk.CreateBillingProfileRequest{
		Name:     "Invoice Override Billing Profile " + prefix,
		Apps:     base.Apps,
		Supplier: supplier,
		Workflow: base.Workflow,
	}
	if edit != nil {
		edit(&request)
	}

	profile, err := c.Billing.CreateProfile(t.Context(), request)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, profile)

	return *profile
}

func manualOverrideFlatFeeRateCard(name string, amount string) v3sdk.RateCardInput {
	cadence := "P1M"
	term := v3sdk.PricePaymentTermInArrears
	price := lo.Must(v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{
		Amount: amount,
	}))

	return v3sdk.RateCardInput{
		Key:            name,
		Name:           name,
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
	}
}

func manualOverrideUnitRateCard(key string, name string, feature v3sdk.Feature, amount string) v3sdk.RateCardInput {
	cadence := "P1M"
	term := v3sdk.PricePaymentTermInArrears
	price := lo.Must(v3sdk.PriceFromPriceUnit(v3sdk.PriceUnit{
		Amount: amount,
	}))

	return v3sdk.RateCardInput{
		Key:            key,
		Name:           name,
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
		Feature:        &v3sdk.FeatureReference{ID: feature.ID},
	}
}

// waitForManualApprovalInvoice returns the first draft invoice whose lines are
// ready for manual approval for a customer. Credit-then-invoice cancellation can
// first surface as a draft waiting for collection, so this helper triggers the
// one available quantity snapshot action before continuing to poll for the
// editable manual-approval state the test mutates.
func waitForManualApprovalInvoice(t *testing.T, client *api.ClientWithResponses, customerID string) api.Invoice {
	t.Helper()

	ctx := t.Context()
	customers := api.InvoiceListParamsCustomers{customerID}
	expand := api.InvoiceListParamsExpand{api.InvoiceExpandLines, api.InvoiceExpandWorkflowApps}
	var invoice api.Invoice
	snapshotQuantitiesRequested := false
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := client.ListInvoicesWithResponse(ctx, &api.ListInvoicesParams{
			Customers: &customers,
			Expand:    &expand,
			PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
		})
		require.NoError(c, err)
		require.Equal(c, http.StatusOK, resp.StatusCode(), "list invoices: %s", string(resp.Body))
		require.NotNil(c, resp.JSON200)

		idx := slices.IndexFunc(resp.JSON200.Items, func(candidate api.Invoice) bool {
			return candidate.Status == api.InvoiceStatusDraft &&
				candidate.StatusDetails.ExtendedStatus == "draft.manual_approval_needed" &&
				candidate.Lines != nil &&
				len(*candidate.Lines) > 0
		})
		if idx != -1 {
			invoice = resp.JSON200.Items[idx]
			return
		}

		if !snapshotQuantitiesRequested {
			idx = slices.IndexFunc(resp.JSON200.Items, func(candidate api.Invoice) bool {
				return candidate.Status == api.InvoiceStatusDraft &&
					candidate.StatusDetails.ExtendedStatus == "draft.waiting_for_collection" &&
					candidate.StatusDetails.AvailableActions.SnapshotQuantities != nil &&
					candidate.Lines != nil &&
					len(*candidate.Lines) > 0
			})
			if idx != -1 {
				snapshotResp, err := client.SnapshotQuantitiesInvoiceActionWithResponse(ctx, resp.JSON200.Items[idx].Id)
				require.NoError(c, err)
				require.Equal(c, http.StatusOK, snapshotResp.StatusCode(), "snapshot quantities: %s", string(snapshotResp.Body))
				require.NotNil(c, snapshotResp.JSON200)

				snapshotQuantitiesRequested = true
			}
		}

		require.Fail(c, "expected a draft invoice waiting for manual approval")
	}, 2*time.Minute, time.Second)

	require.NotEmpty(t, invoice.Id)
	return getInvoiceWithDeletedLines(t, client, invoice.Id)
}

// getInvoiceWithDeletedLines reads invoices with deleted lines included because
// invoice replacement is asserted through the persisted tombstone, not just by
// absence from the active line list.
func getInvoiceWithDeletedLines(t *testing.T, client *api.ClientWithResponses, invoiceID string) api.Invoice {
	t.Helper()

	expand := []api.InvoiceExpand{api.InvoiceExpandLines, api.InvoiceExpandWorkflowApps}
	resp, err := client.GetInvoiceWithResponse(t.Context(), invoiceID, &api.GetInvoiceParams{
		Expand:              &expand,
		IncludeDeletedLines: lo.ToPtr(true),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "get invoice: %s", string(resp.Body))
	require.NotNil(t, resp.JSON200)

	return *resp.JSON200
}

func invoiceLineReplaceUpdateFromLine(t *testing.T, line api.InvoiceLine) api.InvoiceLineReplaceUpdate {
	t.Helper()

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
}

func billingPartyReplaceUpdateFromParty(p api.BillingParty) api.BillingPartyReplaceUpdate {
	return api.BillingPartyReplaceUpdate{
		Addresses: p.Addresses,
		Key:       p.Key,
		Name:      p.Name,
		TaxId:     p.TaxId,
	}
}

func billingPartyReplaceUpdateFromInvoiceCustomer(p api.BillingInvoiceCustomerExtendedDetails) api.BillingPartyReplaceUpdate {
	return api.BillingPartyReplaceUpdate{
		Addresses: p.Addresses,
		Key:       p.Key,
		Name:      p.Name,
		TaxId:     p.TaxId,
	}
}

// invoiceWorkflowReplaceUpdateFromInvoice preserves the current billing
// workflow when replacing invoice lines. The edit endpoint expects a full
// replacement payload, and this test is about line ownership changes rather
// than workflow changes.
func invoiceWorkflowReplaceUpdateFromInvoice(invoice api.Invoice) api.InvoiceWorkflowReplaceUpdate {
	payment := api.BillingWorkflowPaymentSettings{}
	if invoice.Workflow.Workflow.Payment != nil {
		payment = *invoice.Workflow.Workflow.Payment
	}

	invoicing := api.InvoiceWorkflowInvoicingSettingsReplaceUpdate{}
	if invoice.Workflow.Workflow.Invoicing != nil {
		invoicing.AutoAdvance = invoice.Workflow.Workflow.Invoicing.AutoAdvance
		invoicing.DefaultTaxConfig = invoice.Workflow.Workflow.Invoicing.DefaultTaxConfig
		invoicing.DraftPeriod = invoice.Workflow.Workflow.Invoicing.DraftPeriod
		invoicing.DueAfter = invoice.Workflow.Workflow.Invoicing.DueAfter
		invoicing.SubscriptionEndProrationMode = invoice.Workflow.Workflow.Invoicing.SubscriptionEndProrationMode
	}

	return api.InvoiceWorkflowReplaceUpdate{
		Workflow: api.InvoiceWorkflowSettingsReplaceUpdate{
			Invoicing: invoicing,
			Payment:   payment,
		},
	}
}

func flatInvoiceLinePrice(t *testing.T, amount string) api.RateCardUsageBasedPrice {
	t.Helper()

	price := api.RateCardUsageBasedPrice{}
	require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
		Amount:      amount,
		Type:        api.FlatPriceWithPaymentTermTypeFlat,
		PaymentTerm: lo.ToPtr(api.PricePaymentTermInArrears),
	}))

	return price
}

func flatInvoiceLineRateCard(t *testing.T, amount string, source *api.InvoiceUsageBasedRateCard, featureKey *string) *api.InvoiceUsageBasedRateCard {
	t.Helper()

	price := flatInvoiceLinePrice(t, amount)
	rateCard := &api.InvoiceUsageBasedRateCard{
		FeatureKey: featureKey,
		Price:      &price,
	}
	if source != nil {
		rateCard.Discounts = source.Discounts
		if featureKey == nil {
			rateCard.FeatureKey = source.FeatureKey
		}
		rateCard.TaxConfig = source.TaxConfig
	}

	return rateCard
}

func flatInvoiceLineAmount(t *testing.T, line api.InvoiceLine) string {
	t.Helper()

	require.NotNil(t, line.RateCard, "line[%s] rate card", line.Id)
	require.NotNil(t, line.RateCard.Price, "line[%s] rate card price", line.Id)
	flatPrice, err := line.RateCard.Price.AsFlatPriceWithPaymentTerm()
	require.NoError(t, err)

	return flatPrice.Amount
}

func requireInvoiceLineByName(t *testing.T, lines []api.InvoiceLine, name string) api.InvoiceLine {
	t.Helper()

	idx := slices.IndexFunc(lines, func(line api.InvoiceLine) bool {
		return line.Name == name
	})
	require.NotEqual(t, -1, idx, "invoice line %q", name)

	return lines[idx]
}

type invoiceOverrideCharge struct {
	flatFee    *v3sdk.ChargeFlatFee
	usageBased *v3sdk.ChargeUsageBased
}

// listChargesByName groups customer charges by display name while keeping the
// concrete charge shape available. The override assertions need lifecycle
// controller ownership from both flat-fee and usage-based charge variants.
func listChargesByName(t require.TestingT, c *v3Client, customerID string, statuses []v3sdk.ChargeStatus) map[string]invoiceOverrideCharge {
	statusValues := lo.Map(statuses, func(status v3sdk.ChargeStatus, _ int) string {
		return string(status)
	})
	charges, err := c.Customers.Charges.List(c.t.Context(), customerID, v3sdk.ChargeListParams{
		Filter: &v3sdk.ChargeFilter{
			Status: &v3sdk.StringExactFilter{Oeq: statusValues},
		},
		Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
	})
	require.NoError(t, err)
	require.NotNil(t, charges)

	out := make(map[string]invoiceOverrideCharge, len(charges.Data))
	for _, charge := range charges.Data {
		switch charge.Type {
		case string(v3sdk.ChargeTypeFlatFee):
			flatFee, err := charge.AsChargeFlatFee()
			require.NoError(t, err)
			require.NotContains(t, out, flatFee.Name, "duplicate charge name %q", flatFee.Name)
			out[flatFee.Name] = invoiceOverrideCharge{flatFee: flatFee}
		case string(v3sdk.ChargeTypeUsageBased):
			usageBased, err := charge.AsChargeUsageBased()
			require.NoError(t, err)
			require.NotContains(t, out, usageBased.Name, "duplicate charge name %q", usageBased.Name)
			out[usageBased.Name] = invoiceOverrideCharge{usageBased: usageBased}
		}
	}

	return out
}

// LifecycleController reports charge-domain control: manual means an effective
// override layer owns the customer-facing charge behavior.
func assertFlatFeeChargeLifecycleController(t require.TestingT, charges map[string]invoiceOverrideCharge, name string, controller v3sdk.LifecycleController) {
	require.Contains(t, charges, name)
	require.NotNil(t, charges[name].flatFee, "charge %q should be flat-fee", name)
	assert.Equal(t, controller, charges[name].flatFee.LifecycleController, "charge %q lifecycle controller", name)
}

func assertUsageBasedChargeController(t require.TestingT, charges map[string]invoiceOverrideCharge, name string, controller v3sdk.LifecycleController) {
	require.Contains(t, charges, name)
	require.NotNil(t, charges[name].usageBased, "charge %q should be usage-based", name)
	assert.Equal(t, controller, charges[name].usageBased.LifecycleController, "charge %q lifecycle controller", name)
}

func requireFlatFeeChargeIntentMatchesLine(t *testing.T, charges map[string]invoiceOverrideCharge, line api.InvoiceLine) {
	t.Helper()

	require.Contains(t, charges, line.Name)
	charge := charges[line.Name].flatFee
	require.NotNil(t, charge, "charge %q should be flat-fee", line.Name)

	assert.Equal(t, line.Name, charge.Name)
	assert.Equal(t, line.Description, charge.Description)
	assert.Equal(t, "USD", charge.Currency)
	assert.Equal(t, flatInvoiceLineAmount(t, line), charge.AmountAfterProration.Amount)
	assert.Equal(t, lo.FromPtr(line.FeatureKey), lo.FromPtr(charge.FeatureKey))
	assert.Equal(t, line.InvoiceAt, charge.InvoiceAt)
	assert.Equal(t, line.Period.From, charge.ServicePeriod.From, "service period from")
	assert.Equal(t, line.Period.To, charge.ServicePeriod.To, "service period to")

	price, err := charge.Price.AsPriceFlat()
	require.NoError(t, err)
	assert.Equal(t, v3sdk.PriceTypeFlat, price.Type)
	assert.Equal(t, flatInvoiceLineAmount(t, line), price.Amount)
}

func requireFlatFeeChargeSystemIntentMatchesLine(t *testing.T, charges map[string]invoiceOverrideCharge, name string, line api.InvoiceLine, amount string) {
	t.Helper()

	require.Contains(t, charges, name)
	charge := charges[name].flatFee
	require.NotNil(t, charge, "charge %q should be flat-fee", name)
	require.NotNil(t, charge.SystemIntent, "charge %q system intent", name)

	systemIntent := charge.SystemIntent
	assert.Equal(t, line.Name, systemIntent.Name)
	assert.Equal(t, line.Description, systemIntent.Description)
	assert.Equal(t, amount, systemIntent.AmountBeforeProration.Amount)
	assert.Equal(t, line.InvoiceAt, systemIntent.InvoiceAt)
	assert.Equal(t, line.Period.From, systemIntent.ServicePeriod.From, "system intent service period from")
	assert.Equal(t, line.Period.To, systemIntent.ServicePeriod.To, "system intent service period to")
	assert.Equal(t, line.DeletedAt, systemIntent.DeletedAt)
}

func requireFlatFeeChargeHasNoSystemIntent(t *testing.T, charges map[string]invoiceOverrideCharge, name string) {
	t.Helper()

	require.Contains(t, charges, name)
	require.NotNil(t, charges[name].flatFee, "charge %q should be flat-fee", name)
	assert.Nil(t, charges[name].flatFee.SystemIntent, "charge %q system intent", name)
}

func requireUsageBasedChargeHasNoSystemIntent(t *testing.T, charges map[string]invoiceOverrideCharge, name string) {
	t.Helper()

	require.Contains(t, charges, name)
	require.NotNil(t, charges[name].usageBased, "charge %q should be usage-based", name)
	assert.Nil(t, charges[name].usageBased.SystemIntent, "charge %q system intent", name)
}

func requireChargeNames(t *testing.T, charges map[string]invoiceOverrideCharge, expected ...string) {
	t.Helper()

	actual := lo.Keys(charges)
	slices.Sort(actual)
	slices.Sort(expected)
	require.Equal(t, expected, actual)
}

func invoiceLinesCreditsApplied(t *testing.T, lines []api.InvoiceLine) float64 {
	t.Helper()

	return lo.SumBy(lines, func(line api.InvoiceLine) float64 {
		return invoiceLineCreditsApplied(t, line)
	})
}

func activeInvoiceLinesCreditsApplied(t *testing.T, lines []api.InvoiceLine) float64 {
	t.Helper()

	return lo.SumBy(lines, func(line api.InvoiceLine) float64 {
		if line.DeletedAt != nil {
			return 0
		}

		return invoiceLineCreditsApplied(t, line)
	})
}

func invoiceLineCreditsApplied(t *testing.T, line api.InvoiceLine) float64 {
	t.Helper()

	if line.CreditAllocations == nil {
		return 0
	}

	return lo.SumBy(*line.CreditAllocations, func(allocation api.InvoiceLineCreditAllocation) float64 {
		return numericToFloat(t, allocation.Amount)
	})
}

type expectedInvoiceTotals struct {
	Amount              float64
	ChargesTotal        float64
	CreditsTotal        float64
	DiscountsTotal      float64
	TaxesInclusiveTotal float64
	TaxesExclusiveTotal float64
	TaxesTotal          float64
	Total               float64
}

func requireInvoiceTotals(t *testing.T, expected expectedInvoiceTotals, totals api.InvoiceTotals) {
	t.Helper()

	require.Equal(t, expected, expectedInvoiceTotals{
		Amount:              numericToFloat(t, totals.Amount),
		ChargesTotal:        numericToFloat(t, totals.ChargesTotal),
		CreditsTotal:        numericToFloat(t, totals.CreditsTotal),
		DiscountsTotal:      numericToFloat(t, totals.DiscountsTotal),
		TaxesInclusiveTotal: numericToFloat(t, totals.TaxesInclusiveTotal),
		TaxesExclusiveTotal: numericToFloat(t, totals.TaxesExclusiveTotal),
		TaxesTotal:          numericToFloat(t, totals.TaxesTotal),
		Total:               numericToFloat(t, totals.Total),
	})
}

func requireCustomerCreditBalance(t *testing.T, balance *v3sdk.CreditBalances, currency string, expectedSettled float64, expectedPending float64) {
	t.Helper()

	idx := slices.IndexFunc(balance.Balances, func(item v3sdk.CreditBalance) bool {
		return string(item.Currency) == currency
	})
	require.NotEqual(t, -1, idx, "credit balance for %s", currency)

	settled := numericToFloat(t, balance.Balances[idx].Settled)
	pending := numericToFloat(t, balance.Balances[idx].Pending)
	require.Equal(t, expectedSettled, settled, "settled credit balance for %s", currency)
	require.Equal(t, expectedPending, pending, "pending credit balance for %s", currency)
}

func numericToFloat(t require.TestingT, value string) float64 {
	out, err := strconv.ParseFloat(value, 64)
	require.NoError(t, err)
	return out
}

func invoiceOverrideTestKey(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, strings.ToLower(ulid.Make().String()))
}

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
	apiv3 "github.com/openmeterio/openmeter/api/v3"
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

	var customer *apiv3.BillingCustomer
	var plan *apiv3.BillingPlan
	var subscription *apiv3.BillingSubscription
	var invoice api.Invoice
	var deleteLine api.InvoiceLine
	var updateLine api.InvoiceLine
	var keepLine api.InvoiceLine
	var editedInvoice api.Invoice

	runRequired(t, "creates invoice edit plan fixtures", func(t *testing.T) {
		// given:
		// - unique feature and meter keys for this test run
		// when:
		// - a plan is created with three in-arrears flat fees and one usage-based item
		// then:
		// - the plan is published and ready for subscription creation
		status, meter, problem := c.CreateMeter(apiv3.CreateMeterRequest{
			Key:         prefix + "_meter",
			Name:        "Invoice Override Meter " + prefix,
			Aggregation: apiv3.MeterAggregationCount,
			EventType:   prefix + "_event",
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, meter)

		status, meteredFeature, problem := c.CreateFeature(apiv3.CreateFeatureRequest{
			Key:   prefix + "_metered_feature",
			Name:  "Invoice Override Metered Feature " + prefix,
			Meter: &apiv3.FeatureMeterReference{Id: meter.Id},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, meteredFeature)

		phaseKey := "phase_1"
		status, createdPlan, problem := c.CreatePlan(apiv3.CreatePlanRequest{
			Key:              prefix + "_plan",
			Name:             "Invoice Override Plan " + prefix,
			Currency:         "USD",
			BillingCadence:   apiv3.ISO8601Duration("P1M"),
			ProRatingEnabled: lo.ToPtr(false),
			Phases: []apiv3.BillingPlanPhase{{
				Key:  phaseKey,
				Name: "Invoice Override Phase",
				RateCards: []apiv3.BillingRateCard{
					manualOverrideFlatFeeRateCard(flatFeeToDeleteName, "2"),
					manualOverrideFlatFeeRateCard(flatFeeToUpdateName, "5"),
					manualOverrideFlatFeeRateCard(flatFeeToKeepName, "7"),
					manualOverrideUnitRateCard(meteredFeature.Key, usageBasedName, *meteredFeature, "1"),
				},
			}},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, createdPlan)

		status, plan, problem = c.PublishPlan(createdPlan.Id)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)
		require.Equal(t, apiv3.BillingPlanStatusActive, plan.Status)
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

		status, createdCustomer, problem := c.CreateCustomer(apiv3.CreateCustomerRequest{
			Key:      customerKey,
			Name:     "Invoice Override Customer " + prefix,
			Currency: lo.ToPtr(apiv3.CurrencyCode("USD")),
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, createdCustomer)
		customer = createdCustomer

		profile := createNewBillingProfileFromDefault(t, c, prefix, func(profile *apiv3.CreateBillingProfileRequest) {
			profile.Name = "Invoice Override Manual Approval " + prefix
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &apiv3.BillingWorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)

			sendInvoice := apiv3.BillingWorkflowPaymentSettings{}
			require.NoError(t, sendInvoice.FromBillingWorkflowPaymentSendInvoiceSettings(apiv3.BillingWorkflowPaymentSendInvoiceSettings{
				CollectionMethod: apiv3.BillingWorkflowPaymentSendInvoiceSettingsCollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})
		status, _, problem = c.UpdateCustomerBilling(customer.Id, apiv3.UpsertCustomerBillingDataRequest{
			BillingProfile: &apiv3.BillingProfileReference{Id: profile.Id},
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)

		status, _, problem = c.CreateCreditGrant(customer.Id, apiv3.CreateCreditGrantRequest{
			Name:          "Invoice Override Promotional Credits " + prefix,
			Amount:        "100",
			Currency:      apiv3.CreateCurrencyCode("USD"),
			FundingMethod: apiv3.BillingCreditFundingMethodNone,
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)

		status, createdSubscription, problem := c.CreateSubscription(apiv3.BillingSubscriptionCreate{
			Customer: struct {
				Id  *apiv3.ULID                `json:"id,omitempty"`
				Key *apiv3.ExternalResourceKey `json:"key,omitempty"`
			}{
				Id: lo.ToPtr(customer.Id),
			},
			Plan: struct {
				Id      *apiv3.ULID        `json:"id,omitempty"`
				Key     *apiv3.ResourceKey `json:"key,omitempty"`
				Version *int               `json:"version,omitempty"`
			}{
				Id: lo.ToPtr(plan.Id),
			},
			SettlementMode: lo.ToPtr(apiv3.BillingSettlementModeCreditThenInvoice),
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, createdSubscription)
		subscription = createdSubscription

		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			charges := listChargesByName(collect, c, customer.Id, []apiv3.BillingChargeStatus{
				apiv3.BillingChargeStatusCreated,
				apiv3.BillingChargeStatusActive,
				apiv3.BillingChargeStatusFinal,
			})
			assertFlatFeeChargeLifecycleController(collect, charges, flatFeeToDeleteName, apiv3.BillingLifecycleControllerSystem)
			assertFlatFeeChargeLifecycleController(collect, charges, flatFeeToUpdateName, apiv3.BillingLifecycleControllerSystem)
			assertFlatFeeChargeLifecycleController(collect, charges, flatFeeToKeepName, apiv3.BillingLifecycleControllerSystem)
			assertUsageBasedChargeController(collect, charges, usageBasedName, apiv3.BillingLifecycleControllerSystem)
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

		status, canceledSubscription, problem := c.CancelSubscription(subscription.Id, apiv3.BillingSubscriptionCancel{})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, canceledSubscription)
		subscription = canceledSubscription

		invoice = waitForManualApprovalInvoice(t, v1, customer.Id)
		require.NotNil(t, invoice.Lines)
		require.NotEmpty(t, *invoice.Lines)

		linesBeforeEdit := *invoice.Lines
		require.Len(t, linesBeforeEdit, 4)
		deleteLine = requireInvoiceLineByName(t, linesBeforeEdit, flatFeeToDeleteName)
		updateLine = requireInvoiceLineByName(t, linesBeforeEdit, flatFeeToUpdateName)
		keepLine = requireInvoiceLineByName(t, linesBeforeEdit, flatFeeToKeepName)
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

		status, balance, problem := c.GetCustomerCreditBalance(customer.Id)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, balance)
		requireCustomerCreditBalance(t, balance, "USD", 76, 76)

		activeCharges := listChargesByName(t, c, customer.Id, []apiv3.BillingChargeStatus{
			apiv3.BillingChargeStatusCreated,
			apiv3.BillingChargeStatusActive,
			apiv3.BillingChargeStatusFinal,
		})
		requireChargeNames(t, activeCharges, flatFeeToUpdateName, flatFeeToKeepName, createdFlatFeeName, usageBasedName)
		assertFlatFeeChargeLifecycleController(t, activeCharges, flatFeeToUpdateName, apiv3.BillingLifecycleControllerManual)
		requireFlatFeeChargeIntentMatchesLine(t, activeCharges, updatedLineAfterEdit)
		assertFlatFeeChargeLifecycleController(t, activeCharges, flatFeeToKeepName, apiv3.BillingLifecycleControllerSystem)
		requireFlatFeeChargeIntentMatchesLine(t, activeCharges, keepLineAfterEdit)
		assertFlatFeeChargeLifecycleController(t, activeCharges, createdFlatFeeName, apiv3.BillingLifecycleControllerManual)
		requireFlatFeeChargeIntentMatchesLine(t, activeCharges, createdLineAfterEdit)
		assertUsageBasedChargeController(t, activeCharges, usageBasedName, apiv3.BillingLifecycleControllerSystem)

		deletedCharges := listChargesByName(t, c, customer.Id, []apiv3.BillingChargeStatus{
			apiv3.BillingChargeStatusDeleted,
		})
		requireChargeNames(t, deletedCharges, flatFeeToDeleteName)
		assertFlatFeeChargeLifecycleController(t, deletedCharges, flatFeeToDeleteName, apiv3.BillingLifecycleControllerManual)
		requireFlatFeeChargeIntentMatchesLine(t, deletedCharges, deletedLineAfterEdit)
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
func createNewBillingProfileFromDefault(t *testing.T, c *v3Client, prefix string, edit func(*apiv3.CreateBillingProfileRequest)) apiv3.BillingProfile {
	t.Helper()

	status, profiles, problem := c.ListBillingProfiles(withPageSize(100))
	require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
	require.NotNil(t, profiles)

	defaultIdx := slices.IndexFunc(profiles.Data, func(profile apiv3.BillingProfile) bool {
		return profile.Default
	})
	require.NotEqual(t, -1, defaultIdx, "default billing profile is required")

	base := profiles.Data[defaultIdx]
	supplier := base.Supplier
	if supplier.Id != nil && *supplier.Id == "" {
		supplier.Id = nil
	}

	request := apiv3.CreateBillingProfileRequest{
		Name:     "Invoice Override Billing Profile " + prefix,
		Apps:     base.Apps,
		Supplier: supplier,
		Workflow: base.Workflow,
	}
	if edit != nil {
		edit(&request)
	}

	status, profile, problem := c.CreateBillingProfile(request)
	require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
	require.NotNil(t, profile)

	return *profile
}

func manualOverrideFlatFeeRateCard(name string, amount string) apiv3.BillingRateCard {
	cadence := apiv3.ISO8601Duration("P1M")
	term := apiv3.BillingPricePaymentTermInArrears
	price := apiv3.BillingPrice{}
	if err := price.FromBillingPriceFlat(apiv3.BillingPriceFlat{
		Type:   apiv3.BillingPriceFlatTypeFlat,
		Amount: amount,
	}); err != nil {
		panic(err)
	}

	return apiv3.BillingRateCard{
		Key:            name,
		Name:           name,
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
	}
}

func manualOverrideUnitRateCard(key string, name string, feature apiv3.Feature, amount string) apiv3.BillingRateCard {
	cadence := apiv3.ISO8601Duration("P1M")
	term := apiv3.BillingPricePaymentTermInArrears
	price := apiv3.BillingPrice{}
	if err := price.FromBillingPriceUnit(apiv3.BillingPriceUnit{
		Type:   apiv3.BillingPriceUnitTypeUnit,
		Amount: amount,
	}); err != nil {
		panic(err)
	}

	return apiv3.BillingRateCard{
		Key:            key,
		Name:           name,
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
		Feature:        &apiv3.FeatureReferenceItem{Id: feature.Id},
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
	flatFee    *apiv3.BillingChargeFlatFee
	usageBased *apiv3.BillingChargeUsageBased
}

// listChargesByName groups customer charges by display name while keeping the
// concrete charge shape available. The override assertions need lifecycle
// controller ownership from both flat-fee and usage-based charge variants.
func listChargesByName(t require.TestingT, c *v3Client, customerID string, statuses []apiv3.BillingChargeStatus) map[string]invoiceOverrideCharge {
	status, charges, problem := c.ListCustomerChargesByStatus(customerID, statuses, withPageSize(100))
	require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
	require.NotNil(t, charges)

	out := make(map[string]invoiceOverrideCharge, len(charges.Data))
	for _, charge := range charges.Data {
		chargeType, err := charge.Discriminator()
		require.NoError(t, err)

		switch chargeType {
		case "flat_fee":
			flatFee, err := charge.AsBillingChargeFlatFee()
			require.NoError(t, err)
			flatFeeCopy := flatFee
			out[flatFee.Name] = invoiceOverrideCharge{flatFee: &flatFeeCopy}
		case "usage_based":
			usageBased, err := charge.AsBillingChargeUsageBased()
			require.NoError(t, err)
			usageBasedCopy := usageBased
			out[usageBased.Name] = invoiceOverrideCharge{usageBased: &usageBasedCopy}
		}
	}

	return out
}

// LifecycleController is derived from invoice line managedBy ownership, but uses charge-domain values.
func assertFlatFeeChargeLifecycleController(t require.TestingT, charges map[string]invoiceOverrideCharge, name string, controller apiv3.BillingLifecycleController) {
	require.Contains(t, charges, name)
	require.NotNil(t, charges[name].flatFee, "charge %q should be flat-fee", name)
	assert.Equal(t, controller, charges[name].flatFee.LifecycleController, "charge %q lifecycle controller", name)
}

func assertUsageBasedChargeController(t require.TestingT, charges map[string]invoiceOverrideCharge, name string, controller apiv3.BillingLifecycleController) {
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

	price, err := charge.Price.AsBillingPriceFlat()
	require.NoError(t, err)
	assert.Equal(t, apiv3.BillingPriceFlatTypeFlat, price.Type)
	assert.Equal(t, flatInvoiceLineAmount(t, line), price.Amount)
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

func requireCustomerCreditBalance(t *testing.T, balance *apiv3.BillingCreditBalances, currency string, expectedAvailable float64, expectedPending float64) {
	t.Helper()

	idx := slices.IndexFunc(balance.Balances, func(item apiv3.CreditBalance) bool {
		return item.Currency == currency
	})
	require.NotEqual(t, -1, idx, "credit balance for %s", currency)

	available := numericToFloat(t, balance.Balances[idx].Available)
	pending := numericToFloat(t, balance.Balances[idx].Pending)
	require.Equal(t, expectedAvailable, available, "available credit balance for %s", currency)
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

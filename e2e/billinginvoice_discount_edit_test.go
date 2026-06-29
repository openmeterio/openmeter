package e2e

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestInvoiceEditFlatFeeDiscountCreditThenInvoice(t *testing.T) {
	ctx := t.Context()
	client := initClient(t)
	prefix := "discount_edit_" + strings.ToLower(ulid.Make().String())
	lineName := prefix + "_flat_fee"

	customerResp, err := client.CreateCustomerWithResponse(ctx, api.CustomerCreate{
		Key:          lo.ToPtr(prefix + "_customer"),
		Name:         "Invoice Discount Edit Customer " + prefix,
		Currency:     lo.ToPtr(api.CurrencyCode("USD")),
		PrimaryEmail: lo.ToPtr(prefix + "@example.com"),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, customerResp.StatusCode(), "create customer: %s", string(customerResp.Body))
	require.NotNil(t, customerResp.JSON201)

	profileExpand := []api.BillingProfileExpand{api.BillingProfileExpandApps}
	profilesResp, err := client.ListBillingProfilesWithResponse(ctx, &api.ListBillingProfilesParams{
		Expand:   &profileExpand,
		PageSize: lo.ToPtr(api.PaginationPageSize(100)),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, profilesResp.StatusCode(), "list billing profiles: %s", string(profilesResp.Body))
	require.NotNil(t, profilesResp.JSON200)

	defaultProfileIdx := slices.IndexFunc(profilesResp.JSON200.Items, func(profile api.BillingProfile) bool {
		return profile.Default
	})
	require.NotEqual(t, -1, defaultProfileIdx, "default billing profile is required")
	defaultProfile := profilesResp.JSON200.Items[defaultProfileIdx]

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
	require.NotEmpty(t, appRefs.Invoicing.ID, "invoicing app id")
	require.NotEmpty(t, appRefs.Payment.ID, "payment app id")
	require.NotEmpty(t, appRefs.Tax.ID, "tax app id")

	workflow := api.BillingWorkflowCreate{
		Collection: defaultProfile.Workflow.Collection,
		Invoicing:  defaultProfile.Workflow.Invoicing,
		Payment:    defaultProfile.Workflow.Payment,
		Tax:        defaultProfile.Workflow.Tax,
	}
	if workflow.Invoicing == nil {
		workflow.Invoicing = &api.BillingWorkflowInvoicingSettings{}
	}
	workflow.Invoicing.AutoAdvance = lo.ToPtr(false)
	workflow.Payment = &api.BillingWorkflowPaymentSettings{
		CollectionMethod: lo.ToPtr(api.CollectionMethodSendInvoice),
	}

	profileResp, err := client.CreateBillingProfileWithResponse(ctx, api.BillingProfileCreate{
		Name: "Invoice Discount Edit Manual Approval " + prefix,
		Apps: api.BillingProfileAppsCreate{
			Invoicing: appRefs.Invoicing.ID,
			Payment:   appRefs.Payment.ID,
			Tax:       appRefs.Tax.ID,
		},
		Supplier: defaultProfile.Supplier,
		Workflow: workflow,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, profileResp.StatusCode(), "create billing profile: %s", string(profileResp.Body))
	require.NotNil(t, profileResp.JSON201)

	overrideResp, err := client.UpsertBillingProfileCustomerOverrideWithResponse(ctx, customerResp.JSON201.Id, api.BillingProfileCustomerOverrideCreate{
		BillingProfileId: lo.ToPtr(profileResp.JSON201.Id),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, overrideResp.StatusCode(), "upsert billing profile override: %s", string(overrideResp.Body))

	rateCard := api.RateCard{}
	require.NoError(t, rateCard.FromRateCardFlatFee(api.RateCardFlatFee{
		Key:  lineName,
		Name: lineName,
		Type: api.RateCardFlatFeeTypeFlatFee,
		Price: &api.FlatPriceWithPaymentTerm{
			Amount:      "15000",
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		},
		BillingCadence: lo.ToPtr("P1M"),
	}))

	planResp, err := client.CreatePlanWithResponse(ctx, api.PlanCreate{
		Key:            prefix + "_plan",
		Name:           "Invoice Discount Edit Plan " + prefix,
		Currency:       "USD",
		BillingCadence: "P1M",
		SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
		Phases: []api.PlanPhase{{
			Key:       "default",
			Name:      "Default",
			RateCards: []api.RateCard{rateCard},
		}},
		ProRatingConfig: &api.ProRatingConfig{
			Enabled: false,
			Mode:    api.ProRatingModeProratePrices,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, planResp.StatusCode(), "create plan: %s", string(planResp.Body))
	require.NotNil(t, planResp.JSON201)

	publishedPlanResp, err := client.PublishPlanWithResponse(ctx, planResp.JSON201.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, publishedPlanResp.StatusCode(), "publish plan: %s", string(publishedPlanResp.Body))
	require.NotNil(t, publishedPlanResp.JSON200)

	timing := &api.SubscriptionTiming{}
	require.NoError(t, timing.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

	subscriptionCreate := api.SubscriptionCreate{}
	require.NoError(t, subscriptionCreate.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
		Timing:         timing,
		CustomerId:     lo.ToPtr(customerResp.JSON201.Id),
		SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
		Plan: api.PlanReferenceInput{
			Key:     publishedPlanResp.JSON200.Key,
			Version: lo.ToPtr(1),
		},
	}))

	subscriptionResp, err := client.CreateSubscriptionWithResponse(ctx, subscriptionCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, subscriptionResp.StatusCode(), "create subscription: %s", string(subscriptionResp.Body))
	require.NotNil(t, subscriptionResp.JSON201)

	customers := api.InvoiceListParamsCustomers{customerResp.JSON201.Id}
	invoiceExpand := api.InvoiceListParamsExpand{api.InvoiceExpandLines, api.InvoiceExpandWorkflowApps}
	var invoice api.Invoice
	snapshotQuantitiesRequested := false
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := client.ListInvoicesWithResponse(ctx, &api.ListInvoicesParams{
			Customers: &customers,
			Expand:    &invoiceExpand,
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

	getInvoiceExpand := []api.InvoiceExpand{api.InvoiceExpandLines, api.InvoiceExpandWorkflowApps}
	invoiceResp, err := client.GetInvoiceWithResponse(ctx, invoice.Id, &api.GetInvoiceParams{
		Expand: &getInvoiceExpand,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, invoiceResp.StatusCode(), "get invoice: %s", string(invoiceResp.Body))
	require.NotNil(t, invoiceResp.JSON200)
	invoice = *invoiceResp.JSON200
	require.NotNil(t, invoice.Lines)

	lineIdx := slices.IndexFunc(*invoice.Lines, func(line api.InvoiceLine) bool {
		return line.Name == lineName
	})
	require.NotEqual(t, -1, lineIdx, "invoice line %q", lineName)
	lineID := (*invoice.Lines)[lineIdx].Id

	replacementLines := make([]api.InvoiceLineReplaceUpdate, 0, len(*invoice.Lines))
	for _, line := range *invoice.Lines {
		lineUpdate := api.InvoiceLineReplaceUpdate{
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
		if line.Id == lineID {
			require.NotNil(t, lineUpdate.RateCard, "line[%s] rate card", line.Id)
			lineUpdate.RateCard.Discounts = &api.BillingDiscounts{
				Percentage: &api.BillingDiscountPercentage{
					CorrelationId: lo.ToPtr("01G65Z755AFWAKHE12NY0CQ9FH"),
					Percentage:    models.NewPercentage(50),
				},
			}
		}
		replacementLines = append(replacementLines, lineUpdate)
	}

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

	updateResp, err := client.UpdateInvoiceWithResponse(ctx, invoice.Id, api.InvoiceReplaceUpdate{
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
		Workflow: api.InvoiceWorkflowReplaceUpdate{
			Workflow: api.InvoiceWorkflowSettingsReplaceUpdate{
				Invoicing: invoicing,
				Payment:   payment,
			},
		},
		Lines: replacementLines,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, updateResp.StatusCode(), "update invoice: %s", string(updateResp.Body))
	require.NotNil(t, updateResp.JSON200)
}

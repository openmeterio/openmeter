package apiconvert

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestMapTotalsToAPIIncludesCreditsTotal(t *testing.T) {
	got := mapTotalsToAPI(totals.Totals{
		Amount:              alpacadecimal.NewFromInt(100),
		ChargesTotal:        alpacadecimal.NewFromInt(0),
		DiscountsTotal:      alpacadecimal.NewFromInt(10),
		CreditsTotal:        alpacadecimal.NewFromInt(15),
		TaxesInclusiveTotal: alpacadecimal.NewFromInt(0),
		TaxesExclusiveTotal: alpacadecimal.NewFromInt(5),
		TaxesTotal:          alpacadecimal.NewFromInt(5),
		Total:               alpacadecimal.NewFromInt(80),
	})

	require.Equal(t, "15", got.CreditsTotal)
}

func TestMapInvoiceLineCreditsToAPI(t *testing.T) {
	got := mapInvoiceLineCreditsToAPI(billing.CreditsApplied{
		{
			Amount:              alpacadecimal.NewFromFloat(12.34),
			Description:         "credit grant",
			CreditRealizationID: "01K8N36X000000000000000001",
		},
		{
			Amount:              alpacadecimal.NewFromInt(5),
			CreditRealizationID: "01K8N36X000000000000000002",
		},
	})

	require.NotNil(t, got)
	require.Len(t, *got, 2)
	require.Equal(t, "12.34", (*got)[0].Amount)
	require.Equal(t, "credit grant", *(*got)[0].Description)
	require.Equal(t, "5", (*got)[1].Amount)
	require.Nil(t, (*got)[1].Description)
}

func TestMapInvoiceLineCreditsToAPIOmitsEmptyCredits(t *testing.T) {
	require.Nil(t, mapInvoiceLineCreditsToAPI(nil))
}

func TestMapSubscriptionReferencesToAPIKeepsIdsOnDocumentedFields(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)

	got := ToAPIInvoiceLineSubscriptionReference(&billing.SubscriptionReference{
		SubscriptionID: "01M6QNHXFHPA5QN3TGVPYFHZPR",
		PhaseID:        "01PHASE0000000000000000001",
		ItemID:         "01ITEM00000000000000000001",
		BillingPeriod: timeutil.ClosedPeriod{
			From: from,
			To:   to,
		},
	})

	require.NotNil(t, got)
	require.Equal(t, "01M6QNHXFHPA5QN3TGVPYFHZPR", got.Subscription.Id)
	require.Equal(t, "01PHASE0000000000000000001", got.Phase.Id)
	require.Equal(t, "01ITEM00000000000000000001", got.Item.Id)
	require.True(t, from.Equal(got.BillingPeriod.From))
	require.True(t, to.Equal(got.BillingPeriod.To))
	require.Nil(t, ToAPIInvoiceLineSubscriptionReference(nil))
}

// mainFormatInvoiceEvent is an invoice event as the production code published it before lines
// carried the invoicing app tax code: the line's tax config sits under taxOverrides with the
// Stripe code in it, and there is no invoicingAppTaxCode.
const mainFormatInvoiceEvent = `{
  "invoice": {
    "id": "01JINVOICE000000000000000A",
    "namespace": "ns",
    "number": "INV-1",
    "type": "standard",
    "currency": "USD",
    "status": "draft.created",
    "createdAt": "2026-01-01T00:00:00Z",
    "updatedAt": "2026-01-01T00:00:00Z",
    "customer": {"customerId": "01JCUSTOMER0000000000000A", "name": "Test Customer"},
    "workflow": {"config": {"collection": {"alignment": "subscription"}}},
    "lines": [{
      "id": "01JLINE0000000000000000000A",
      "namespace": "ns",
      "name": "line",
      "createdAt": "2026-01-01T00:00:00Z",
      "updatedAt": "2026-01-01T00:00:00Z",
      "managedBy": "manual",
      "engine": "invoice",
      "currency": "USD",
      "period": {"from": "2026-01-01T00:00:00Z", "to": "2026-02-01T00:00:00Z"},
      "invoiceAt": "2026-02-01T00:00:00Z",
      "taxOverrides": {
        "behavior": "exclusive",
        "stripe": {"code": "txcd_main"},
        "tax_code_id": "01JTAXCODE0000000000000000A",
        "tax_code": {"namespace": "ns", "id": "01JTAXCODE0000000000000000A", "key": "txcd_10000000", "name": "General"}
      },
      "usageBased": {"price": {"type": "flat", "amount": "10", "paymentTerm": "in_advance"}}
    }]
  },
  "apps": {
    "invoicing": {
      "namespace": "ns", "id": "01JAPP00000000000000000001", "name": "Stripe", "type": "stripe",
      "status": "ready", "listing": {"type": "stripe"}, "appData": {}
    }
  }
}`

func TestToAPIEventInvoiceStripeCodeFromEventLines(t *testing.T) {
	const taxCodeID = "01JTAXCODE0000000000000000A"

	// The live mapping differs from every code carried by the events below.
	taxCodes := productcatalog.TaxCodesByID{taxCodeID: {
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: taxCodeID},
		AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_live"}},
	}}

	t.Run("an event from before the stamp shows the stripe code of its line taxOverrides", func(t *testing.T) {
		// given: a main-format event JSON, decoded like the notification consumer does
		var event billing.StandardInvoiceCreatedEvent
		require.NoError(t, json.Unmarshal([]byte(mainFormatInvoiceEvent), &event))

		line := event.Invoice.Lines.OrEmpty()[0]
		require.Nil(t, line.InvoicingAppTaxCode)
		require.Equal(t, "txcd_main", line.LegacyStripeTaxCode)

		// when: the event is rendered to the v1 invoice
		got, err := ToAPIEventInvoice(event.EventStandardInvoice, taxCodes)
		require.NoError(t, err)

		// then: the line shows the code it was billed with, not the live mapping
		require.NotNil(t, got.Lines)
		require.Len(t, *got.Lines, 1)
		require.Equal(t, &apiv1models.StripeTaxConfig{Code: "txcd_main"}, (*got.Lines)[0].TaxConfig.Stripe)
	})

	t.Run("an event with a stamped line shows the stamp even when the live mapping differs", func(t *testing.T) {
		// given: a new event, built from the same invoice with the line stamped, and sent over JSON
		var old billing.StandardInvoiceCreatedEvent
		require.NoError(t, json.Unmarshal([]byte(mainFormatInvoiceEvent), &old))

		invoice := old.Invoice
		invoice.Lines.OrEmpty()[0].InvoicingAppTaxCode = lo.ToPtr("txcd_stamped")

		built, err := billing.NewStandardInvoiceCreatedEvent(invoice, taxCodes)
		require.NoError(t, err)

		built.Apps = old.Apps

		raw, err := json.Marshal(built)
		require.NoError(t, err)

		var event billing.StandardInvoiceCreatedEvent
		require.NoError(t, json.Unmarshal(raw, &event))

		// when: the event is rendered to the v1 invoice
		got, err := ToAPIEventInvoice(event.EventStandardInvoice, taxCodes)
		require.NoError(t, err)

		// then: the line shows the stamp, not the live mapping
		require.NotNil(t, got.Lines)
		require.Len(t, *got.Lines, 1)
		require.Equal(t, &apiv1models.StripeTaxConfig{Code: "txcd_stamped"}, (*got.Lines)[0].TaxConfig.Stripe)
	})
}

// eventTestApp is the smallest app the invoice event constructors accept: they only read its
// base and event data.
type eventTestApp struct {
	app.App

	base app.AppBase
}

func (a eventTestApp) GetAppBase() app.AppBase { return a.base }

func (a eventTestApp) GetType() app.AppType { return a.base.Type }

func (a eventTestApp) GetEventAppData() (app.EventAppData, error) {
	return app.NewEventAppData(map[string]any{})
}

func TestToAPIEventInvoiceTaxOverridesStripeCode(t *testing.T) {
	const taxCodeID = "01JTAXCODE0000000000000000A"

	// The live mapping differs from every code a line was billed with below.
	taxCodes := productcatalog.TaxCodesByID{taxCodeID: {
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: taxCodeID},
		AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_live"}},
	}}

	tests := []struct {
		name             string
		invoicingAppType app.AppType
		stamp            *string
		legacyCode       string

		// wantEventCode is the stripe.code in the JSON of the event, nil for no stripe object.
		wantEventCode *string
		// wantRenderedCode is the stripe.code of the line in the rendered notification.
		wantRenderedCode *string
	}{
		{
			name:             "an unstamped line sends the code it was billed with, not the live mapping",
			invoicingAppType: app.AppTypeStripe,
			legacyCode:       "txcd_billed",
			wantEventCode:    lo.ToPtr("txcd_billed"),
			wantRenderedCode: lo.ToPtr("txcd_billed"),
		},
		{
			name:             "an unstamped line billed without a code sends no stripe object",
			invoicingAppType: app.AppTypeStripe,
			wantEventCode:    nil,
			wantRenderedCode: nil,
		},
		{
			name:             "a stamped line sends its stamp",
			invoicingAppType: app.AppTypeStripe,
			stamp:            lo.ToPtr("txcd_stamped"),
			legacyCode:       "txcd_billed",
			wantEventCode:    lo.ToPtr("txcd_stamped"),
			wantRenderedCode: lo.ToPtr("txcd_stamped"),
		},
		{
			name:             "a line stamped without a code sends no stripe object",
			invoicingAppType: app.AppTypeStripe,
			stamp:            lo.ToPtr(""),
			legacyCode:       "txcd_billed",
			wantEventCode:    nil,
			wantRenderedCode: nil,
		},
		{
			name:             "a non-stripe invoice keeps the live mapping",
			invoicingAppType: app.AppTypeSandbox,
			legacyCode:       "txcd_billed",
			wantEventCode:    lo.ToPtr("txcd_live"),
			wantRenderedCode: lo.ToPtr("txcd_live"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an issued invoice with one line billed as described by the case
			var base billing.StandardInvoiceCreatedEvent
			require.NoError(t, json.Unmarshal([]byte(mainFormatInvoiceEvent), &base))

			invoice := base.Invoice
			invoice.Status = billing.StandardInvoiceStatusIssued

			line := invoice.Lines.OrEmpty()[0]
			line.TaxConfig = &productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID}
			line.InvoicingAppTaxCode = tt.stamp
			line.LegacyStripeTaxCode = tt.legacyCode

			invoicingApp := eventTestApp{base: app.AppBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: "ns",
					ID:        "01JAPP00000000000000000001",
					Name:      "Invoicing",
				}),
				Type:   tt.invoicingAppType,
				Status: app.AppStatusReady,
			}}
			invoice.Workflow.Apps = &billing.ProfileApps{
				Tax:       invoicingApp,
				Invoicing: invoicingApp,
				Payment:   invoicingApp,
			}

			// when: the event is built, sent over JSON and rendered like a payment update
			built, err := billing.NewStandardInvoiceUpdatedEvent(billing.StandardInvoiceUpdatedEventInput{
				New:      invoice,
				Old:      base.EventStandardInvoice,
				TaxCodes: taxCodes,
			})
			require.NoError(t, err)

			raw, err := json.Marshal(built.New)
			require.NoError(t, err)

			var event billing.EventStandardInvoice
			require.NoError(t, json.Unmarshal(raw, &event))

			got, err := ToAPIEventInvoice(event, taxCodes)
			require.NoError(t, err)

			// then: the JSON and the rendering both carry the expected code
			var wire struct {
				Invoice struct {
					Lines []struct {
						TaxOverrides struct {
							Stripe *struct {
								Code string `json:"code"`
							} `json:"stripe"`
						} `json:"taxOverrides"`
					} `json:"lines"`
				} `json:"invoice"`
			}
			require.NoError(t, json.Unmarshal(raw, &wire))
			require.Len(t, wire.Invoice.Lines, 1)

			var eventCode *string
			if stripe := wire.Invoice.Lines[0].TaxOverrides.Stripe; stripe != nil {
				eventCode = &stripe.Code
			}

			require.Equal(t, tt.wantEventCode, eventCode)

			require.NotNil(t, got.Lines)
			require.Len(t, *got.Lines, 1)

			var renderedCode *string
			if stripe := (*got.Lines)[0].TaxConfig.Stripe; stripe != nil {
				renderedCode = &stripe.Code
			}

			require.Equal(t, tt.wantRenderedCode, renderedCode)
		})
	}
}

func TestToAPIStandardInvoiceGatheringPreviewShowsLiveStripeCode(t *testing.T) {
	const taxCodeID = "01JTAXCODE0000000000000000A"

	taxCodes := productcatalog.TaxCodesByID{taxCodeID: {
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: taxCodeID},
		AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_live"}},
	}}

	tests := []struct {
		name   string
		status billing.StandardInvoiceStatus
		// wantCode is the stripe.code every rendering of the line shows, "" for no stripe object.
		wantCode string
	}{
		{
			name:     "a gathering invoice shows the live mapping",
			status:   billing.StandardInvoiceStatusGathering,
			wantCode: "txcd_live",
		},
		{
			// The line was never stamped and has no legacy code, so it was billed without one.
			name:     "an issued invoice shows the billed code, which is none",
			status:   billing.StandardInvoiceStatusIssued,
			wantCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a Stripe invoice with an unstamped line with a detailed line
			var base billing.StandardInvoiceCreatedEvent
			require.NoError(t, json.Unmarshal([]byte(mainFormatInvoiceEvent), &base))

			invoice := base.Invoice
			invoice.Status = tt.status

			line := invoice.Lines.OrEmpty()[0]
			line.TaxConfig = &productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID}
			line.InvoicingAppTaxCode = nil
			line.LegacyStripeTaxCode = ""
			line.DetailedLines = billing.DetailedLines{{}}

			// when: the invoice is mapped to v1
			got, err := ToAPIStandardInvoice(invoice, taxCodes, app.AppTypeStripe)
			require.NoError(t, err)

			// then: the line, its rate card and its detailed lines agree
			var want *apiv1models.StripeTaxConfig
			if tt.wantCode != "" {
				want = &apiv1models.StripeTaxConfig{Code: tt.wantCode}
			}

			require.NotNil(t, got.Lines)
			require.Len(t, *got.Lines, 1)

			apiLine := (*got.Lines)[0]
			require.Equal(t, want, apiLine.TaxConfig.Stripe, "line")
			require.Equal(t, want, apiLine.RateCard.TaxConfig.Stripe, "rate card")

			require.NotNil(t, apiLine.Children)
			require.Len(t, *apiLine.Children, 1)
			require.Equal(t, want, (*apiLine.Children)[0].TaxConfig.Stripe, "detailed line")
		})
	}
}

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

		// when
		got, err := ToAPIEventInvoice(event.EventStandardInvoice, taxCodes)
		require.NoError(t, err)

		// then
		require.NotNil(t, got.Lines)
		require.Len(t, *got.Lines, 1)
		require.Equal(t, &apiv1models.StripeTaxConfig{Code: "txcd_stamped"}, (*got.Lines)[0].TaxConfig.Stripe)
	})
}

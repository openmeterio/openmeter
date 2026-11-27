package billing

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// The raw events keep the legacy `taxOverrides` line shape for consumers of the system events
// topic, even though domain lines only carry the Tax Code ID. These tests build events directly
// (no DB) and assert on the serialized JSON.

const (
	eventTestNamespace = "ns"
	eventTestTaxCodeID = "01JTAXCODE00000000000000000"
)

func eventTestTaxCode(withStripe bool) *taxcode.TaxCode {
	tc := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: eventTestNamespace, ID: eventTestTaxCodeID},
		Key:          "txcd_10000000",
		Name:         "General",
	}

	if withStripe {
		tc.AppMappings = taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}}
	}

	return tc
}

func eventTestTaxConfig() *productcatalog.TaxCodeConfig {
	return &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: eventTestTaxCodeID,
	}
}

func eventTestStandardInvoice(cfg *productcatalog.TaxCodeConfig) StandardInvoice {
	now := time.Now().UTC().Truncate(time.Second)

	line := NewFlatFeeLine(NewFlatFeeLineInput{
		Namespace: eventTestNamespace,
		ID:        "line-1",
		CreatedAt: now,
		UpdatedAt: now,
		Name:      "line",
	})
	line.TaxConfig = cfg

	return StandardInvoice{
		StandardInvoiceBase: StandardInvoiceBase{Namespace: eventTestNamespace, ID: "inv-1"},
		Lines:               NewStandardInvoiceLines([]*StandardLine{line}),
	}
}

func eventTestGatheringInvoice(cfg *productcatalog.TaxCodeConfig) GatheringInvoice {
	now := time.Now().UTC().Truncate(time.Second)

	line := GatheringLine{
		GatheringLineBase: GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: eventTestNamespace,
				ID:        "line-1",
				CreatedAt: now,
				UpdatedAt: now,
				Name:      "line",
			}),
			Price:     *productcatalog.NewPriceFrom(productcatalog.FlatPrice{}),
			TaxConfig: cfg,
		},
	}

	return GatheringInvoice{
		GatheringInvoiceBase: GatheringInvoiceBase{Namespace: eventTestNamespace, ID: "gi-1"},
		Lines:                GatheringInvoiceLines{Option: mo.Some(GatheringLines{line})},
	}
}

func eventTestDecode(t *testing.T, v any) map[string]any {
	t.Helper()

	raw, err := json.Marshal(v)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))

	return out
}

func eventTestFirstLine(t *testing.T, invoice any) map[string]any {
	t.Helper()

	m, ok := invoice.(map[string]any)
	require.True(t, ok)

	lines, ok := m["lines"].([]any)
	require.True(t, ok)
	require.Len(t, lines, 1)

	line, ok := lines[0].(map[string]any)
	require.True(t, ok)

	return line
}

func TestEventStandardInvoiceLegacyTaxOverrides(t *testing.T) {
	taxCodes := map[string]*taxcode.TaxCode{eventTestTaxCodeID: eventTestTaxCode(true)}

	t.Run("a line referencing a Tax Code that is not loaded is a validation error", func(t *testing.T) {
		// given: a standard line whose Tax Code is missing from the loaded Tax Codes
		// when: the event is constructed
		_, err := NewEventStandardInvoice(eventTestStandardInvoice(eventTestTaxConfig()), productcatalog.TaxCodesByID{})

		// then: construction fails before anything is built, naming the reference
		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, eventTestTaxCodeID)
	})

	t.Run("created event renders the legacy taxOverrides", func(t *testing.T) {
		// given: a standard line referencing a Tax Code with a Stripe mapping
		// when: the created event is serialized
		event, err := NewStandardInvoiceCreatedEvent(eventTestStandardInvoice(eventTestTaxConfig()), taxCodes)
		require.NoError(t, err)

		out := eventTestDecode(t, event)

		// then: the top-level keys are unchanged and the line carries the full legacy shape
		require.Equal(t, []string{"apps", "invoice"}, keysOf(out))

		overrides, ok := eventTestFirstLine(t, out["invoice"])["taxOverrides"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "inclusive", overrides["behavior"])
		require.Equal(t, eventTestTaxCodeID, overrides["tax_code_id"])
		require.Equal(t, map[string]any{"code": "txcd_10000000"}, overrides["stripe"])

		tc, ok := overrides["tax_code"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, eventTestTaxCodeID, tc["id"])
	})

	t.Run("updated event renders both old and new", func(t *testing.T) {
		// given: old and new states referencing the same Tax Code
		invoice := eventTestStandardInvoice(eventTestTaxConfig())

		old, err := NewEventStandardInvoice(invoice, taxCodes)
		require.NoError(t, err)

		// when: the updated event is serialized
		event, err := NewStandardInvoiceUpdatedEvent(StandardInvoiceUpdatedEventInput{New: invoice, Old: old, TaxCodes: taxCodes})
		require.NoError(t, err)

		out := eventTestDecode(t, event)

		// then: both sides carry taxOverrides
		for _, side := range []string{"old", "new"} {
			sideMap, ok := out[side].(map[string]any)
			require.True(t, ok)

			overrides, ok := eventTestFirstLine(t, sideMap["invoice"])["taxOverrides"].(map[string]any)
			require.True(t, ok, side)
			require.Equal(t, eventTestTaxCodeID, overrides["tax_code_id"])
			require.Contains(t, overrides, "tax_code")
		}
	})

	t.Run("tax code without Stripe mapping has no stripe key", func(t *testing.T) {
		// given: a Tax Code with no Stripe app mapping
		noStripe := map[string]*taxcode.TaxCode{eventTestTaxCodeID: eventTestTaxCode(false)}

		// when
		event, err := NewStandardInvoiceCreatedEvent(eventTestStandardInvoice(eventTestTaxConfig()), noStripe)
		require.NoError(t, err)

		// then
		overrides, ok := eventTestFirstLine(t, eventTestDecode(t, event)["invoice"])["taxOverrides"].(map[string]any)
		require.True(t, ok)
		require.NotContains(t, overrides, "stripe")
		require.Contains(t, overrides, "tax_code")
	})

	t.Run("nil line config omits taxOverrides", func(t *testing.T) {
		// given: a line without tax config
		event, err := NewStandardInvoiceCreatedEvent(eventTestStandardInvoice(nil), nil)
		require.NoError(t, err)

		// when/then: the key is absent, not null
		require.NotContains(t, eventTestFirstLine(t, eventTestDecode(t, event)["invoice"]), "taxOverrides")
	})

	t.Run("missing tax code fails the constructor", func(t *testing.T) {
		// given: a line referencing a Tax Code that was not loaded
		// when
		_, err := NewStandardInvoiceCreatedEvent(eventTestStandardInvoice(eventTestTaxConfig()), map[string]*taxcode.TaxCode{})

		// then
		require.ErrorContains(t, err, eventTestTaxCodeID)
	})

	t.Run("round trip restores the domain tax config", func(t *testing.T) {
		// given: a serialized created event
		event, err := NewStandardInvoiceCreatedEvent(eventTestStandardInvoice(eventTestTaxConfig()), taxCodes)
		require.NoError(t, err)

		raw, err := json.Marshal(event)
		require.NoError(t, err)

		// when: decoded into the event type
		var decoded StandardInvoiceCreatedEvent
		require.NoError(t, json.Unmarshal(raw, &decoded))

		// then: the legacy extras are ignored and the domain config survives
		require.Equal(t, eventTestTaxConfig(), decoded.Invoice.Lines.OrEmpty()[0].TaxConfig)
	})
}

func TestEventGatheringInvoiceLegacyTaxOverrides(t *testing.T) {
	taxCodes := map[string]*taxcode.TaxCode{eventTestTaxCodeID: eventTestTaxCode(true)}

	t.Run("gathering lines carry stripe but no tax_code", func(t *testing.T) {
		// given: a gathering line referencing a Tax Code with a Stripe mapping
		event, err := NewGatheringInvoiceCreatedEvent(eventTestGatheringInvoice(eventTestTaxConfig()), taxCodes)
		require.NoError(t, err)

		// when
		out := eventTestDecode(t, event)

		// then
		require.Equal(t, []string{"gatheringInvoice"}, keysOf(out))

		overrides, ok := eventTestFirstLine(t, out["gatheringInvoice"])["taxOverrides"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "inclusive", overrides["behavior"])
		require.Equal(t, eventTestTaxCodeID, overrides["tax_code_id"])
		require.Equal(t, map[string]any{"code": "txcd_10000000"}, overrides["stripe"])
		require.NotContains(t, overrides, "tax_code")
	})

	t.Run("nil line config omits taxOverrides", func(t *testing.T) {
		event, err := NewGatheringInvoiceCreatedEvent(eventTestGatheringInvoice(nil), nil)
		require.NoError(t, err)

		require.NotContains(t, eventTestFirstLine(t, eventTestDecode(t, event)["gatheringInvoice"]), "taxOverrides")
	})

	t.Run("missing tax code fails the constructor", func(t *testing.T) {
		_, err := NewGatheringInvoiceCreatedEvent(eventTestGatheringInvoice(eventTestTaxConfig()), nil)

		require.ErrorContains(t, err, eventTestTaxCodeID)
	})

	t.Run("round trip restores the domain tax config", func(t *testing.T) {
		event, err := NewGatheringInvoiceCreatedEvent(eventTestGatheringInvoice(eventTestTaxConfig()), taxCodes)
		require.NoError(t, err)

		raw, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded GatheringInvoiceCreatedEvent
		require.NoError(t, json.Unmarshal(raw, &decoded))

		require.Equal(t, eventTestTaxConfig(), decoded.Invoice.Lines.OrEmpty()[0].TaxConfig)
	})
}

func TestEventStandardInvoiceReencode(t *testing.T) {
	taxCodes := map[string]*taxcode.TaxCode{eventTestTaxCodeID: eventTestTaxCode(true)}

	t.Run("a decoded created event encodes to the same JSON and keeps the stamp", func(t *testing.T) {
		// given: a created event with a stamped line referencing a Tax Code
		invoice := eventTestStandardInvoice(eventTestTaxConfig())
		invoice.Lines.OrEmpty()[0].InvoicingAppTaxCode = lo.ToPtr("txcd_stamped")

		event, err := NewStandardInvoiceCreatedEvent(invoice, taxCodes)
		require.NoError(t, err)

		raw, err := json.Marshal(event)
		require.NoError(t, err)

		// when: it is decoded and encoded again
		var decoded StandardInvoiceCreatedEvent
		require.NoError(t, json.Unmarshal(raw, &decoded))

		again, err := json.Marshal(decoded)
		require.NoError(t, err)

		// then: the JSON is identical and the stamp and the legacy stripe code survive
		require.Equal(t, string(raw), string(again))

		line := decoded.Invoice.Lines.OrEmpty()[0]
		require.Equal(t, lo.ToPtr("txcd_stamped"), line.InvoicingAppTaxCode)
		require.Equal(t, "txcd_10000000", line.LegacyStripeTaxCode)
		require.Contains(t, string(raw), `"invoicingAppTaxCode":"txcd_stamped"`)
		require.NotContains(t, string(raw), "legacyStripeTaxCode")
	})

	t.Run("a decoded updated event encodes to the same JSON on both sides", func(t *testing.T) {
		// given: an updated event whose old and new sides reference the Tax Code
		invoice := eventTestStandardInvoice(eventTestTaxConfig())
		invoice.Lines.OrEmpty()[0].InvoicingAppTaxCode = lo.ToPtr("")

		old, err := NewEventStandardInvoice(invoice, taxCodes)
		require.NoError(t, err)

		event, err := NewStandardInvoiceUpdatedEvent(StandardInvoiceUpdatedEventInput{New: invoice, Old: old, TaxCodes: taxCodes})
		require.NoError(t, err)

		raw, err := json.Marshal(event)
		require.NoError(t, err)

		// when: it is decoded and encoded again
		var decoded StandardInvoiceUpdatedEvent
		require.NoError(t, json.Unmarshal(raw, &decoded))

		again, err := json.Marshal(decoded)
		require.NoError(t, err)

		// then: both sides round-trip and the empty stamp stays an empty stamp
		require.Equal(t, string(raw), string(again))
		require.Equal(t, lo.ToPtr(""), decoded.Old.Invoice.Lines.OrEmpty()[0].InvoicingAppTaxCode)
		require.Equal(t, lo.ToPtr(""), decoded.New.Invoice.Lines.OrEmpty()[0].InvoicingAppTaxCode)
	})

	t.Run("a decoded gathering event encodes to the same JSON", func(t *testing.T) {
		// given: a created event for a gathering invoice whose line references a Tax Code
		event, err := NewGatheringInvoiceCreatedEvent(eventTestGatheringInvoice(eventTestTaxConfig()), taxCodes)
		require.NoError(t, err)

		raw, err := json.Marshal(event)
		require.NoError(t, err)

		// when: it is decoded and encoded again
		var decoded GatheringInvoiceCreatedEvent
		require.NoError(t, json.Unmarshal(raw, &decoded))

		again, err := json.Marshal(decoded)
		require.NoError(t, err)

		// then: the JSON is identical
		require.Equal(t, string(raw), string(again))
	})

	t.Run("an event built without a constructor encodes the Tax Code ID and behavior only", func(t *testing.T) {
		// given: invoices whose lines reference a Tax Code nobody loaded
		invoice := eventTestStandardInvoice(eventTestTaxConfig())

		// when: events built as struct literals, without a constructor, are encoded
		standard, err := json.Marshal(EventStandardInvoice{Invoice: invoice})
		require.NoError(t, err)

		updated, err := json.Marshal(StandardInvoiceUpdatedEvent{
			Old: EventStandardInvoice{Invoice: invoice},
			New: EventStandardInvoice{Invoice: invoice},
		})
		require.NoError(t, err)

		gathering, err := json.Marshal(GatheringInvoiceCreatedEvent{Invoice: eventTestGatheringInvoice(eventTestTaxConfig())})
		require.NoError(t, err)

		// then: encoding does not fail and the legacy payload has no looked-up parts
		wantOverrides := map[string]any{"behavior": "inclusive", "tax_code_id": eventTestTaxCodeID}

		for name, payload := range map[string][]byte{"standard": standard, "gathering": gathering} {
			var out map[string]any
			require.NoError(t, json.Unmarshal(payload, &out), name)

			key := "invoice"
			if name == "gathering" {
				key = "gatheringInvoice"
			}

			require.Equal(t, wantOverrides, eventTestFirstLine(t, out[key])["taxOverrides"], name)
		}

		var out map[string]any
		require.NoError(t, json.Unmarshal(updated, &out))

		for _, side := range []string{"old", "new"} {
			sideMap, ok := out[side].(map[string]any)
			require.True(t, ok)
			require.Equal(t, wantOverrides, eventTestFirstLine(t, sideMap["invoice"])["taxOverrides"], side)
		}
	})

	t.Run("a decoded event without a legacy stripe code leaves it empty", func(t *testing.T) {
		// given: a Tax Code without a Stripe mapping
		noStripe := map[string]*taxcode.TaxCode{eventTestTaxCodeID: eventTestTaxCode(false)}

		event, err := NewStandardInvoiceCreatedEvent(eventTestStandardInvoice(eventTestTaxConfig()), noStripe)
		require.NoError(t, err)

		raw, err := json.Marshal(event)
		require.NoError(t, err)

		// when: it is decoded
		var decoded StandardInvoiceCreatedEvent
		require.NoError(t, json.Unmarshal(raw, &decoded))

		// then: the line has no legacy stripe code and no stamp
		line := decoded.Invoice.Lines.OrEmpty()[0]
		require.Empty(t, line.LegacyStripeTaxCode)
		require.Nil(t, line.InvoicingAppTaxCode)
	})

	t.Run("a decoded event gives each line its own legacy stripe code", func(t *testing.T) {
		// given: an event with a line on a Stripe-mapped Tax Code, a line on a Tax Code with
		// another mapping, and a line without a Tax Code
		otherTaxCode := &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{Namespace: eventTestNamespace, ID: "01JTAXCODE00000000000000001"},
			Key:          "txcd_20000000",
			Name:         "Other",
			AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_20000000"}},
		}
		twoTaxCodes := map[string]*taxcode.TaxCode{
			eventTestTaxCodeID: eventTestTaxCode(true),
			otherTaxCode.ID:    otherTaxCode,
		}

		invoice := eventTestStandardInvoice(eventTestTaxConfig())
		now := time.Now().UTC().Truncate(time.Second)

		other := NewFlatFeeLine(NewFlatFeeLineInput{Namespace: eventTestNamespace, ID: "line-2", CreatedAt: now, UpdatedAt: now, Name: "other"})
		other.TaxConfig = &productcatalog.TaxCodeConfig{TaxCodeID: otherTaxCode.ID}

		untaxed := NewFlatFeeLine(NewFlatFeeLineInput{Namespace: eventTestNamespace, ID: "line-3", CreatedAt: now, UpdatedAt: now, Name: "untaxed"})

		invoice.Lines = NewStandardInvoiceLines(append(invoice.Lines.OrEmpty(), other, untaxed))

		event, err := NewStandardInvoiceCreatedEvent(invoice, twoTaxCodes)
		require.NoError(t, err)

		raw, err := json.Marshal(event)
		require.NoError(t, err)

		// when: it is decoded
		var decoded StandardInvoiceCreatedEvent
		require.NoError(t, json.Unmarshal(raw, &decoded))

		// then: every line has the stripe code of its own taxOverrides, matched by line ID
		codes := make(map[string]string)
		for _, line := range decoded.Invoice.Lines.OrEmpty() {
			codes[line.ID] = line.LegacyStripeTaxCode
		}

		require.Equal(t, map[string]string{
			"line-1": "txcd_10000000",
			"line-2": "txcd_20000000",
			"line-3": "",
		}, codes)
	})
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	return keys
}

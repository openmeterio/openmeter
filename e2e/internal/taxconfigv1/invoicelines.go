package taxconfigv1

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
)

// LineInput is one pending invoice line to create or update. v1 accepts a line's tax
// config in rateCard.taxConfig, or in the deprecated top-level taxConfig next to a top-level
// price; the input holds either shape.
type LineInput struct {
	Name      string
	InvoiceAt time.Time
	Period    api.Period
	RateCard  *api.InvoiceUsageBasedRateCard
	Price     *api.RateCardUsageBasedPrice
	TaxConfig *api.TaxConfig
}

// FlatLinePrice builds the flat fee price of a line input, fresh per call so line
// inputs don't share one price value.
func FlatLinePrice(t *testing.T) *api.RateCardUsageBasedPrice {
	t.Helper()

	price := api.RateCardUsageBasedPrice{}
	require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
		Amount:      "10",
		Type:        api.FlatPriceWithPaymentTermTypeFlat,
		PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
	}))

	return &price
}

// UnitLinePrice builds the unit price of a usage based line input, fresh per call so
// line inputs don't share one price value.
func UnitLinePrice(t *testing.T) *api.RateCardUsageBasedPrice {
	t.Helper()

	price := api.RateCardUsageBasedPrice{}
	require.NoError(t, price.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
		Amount: "0.1",
		Type:   api.UnitPriceWithCommitmentsTypeUnit,
	}))

	return &price
}

// PendingLines builds the pending line create requests for the given inputs.
func PendingLines(inputs []LineInput) []api.InvoicePendingLineCreate {
	return lo.Map(inputs, func(in LineInput, _ int) api.InvoicePendingLineCreate {
		return api.InvoicePendingLineCreate{
			Name:      in.Name,
			InvoiceAt: in.InvoiceAt,
			Period:    in.Period,
			RateCard:  in.RateCard,
			Price:     in.Price,
			TaxConfig: in.TaxConfig,
		}
	})
}

// NewInvoiceLines builds the replace updates for creating the given inputs as new
// lines of an invoice edit.
func NewInvoiceLines(inputs []LineInput) []api.InvoiceLineReplaceUpdate {
	return lo.Map(inputs, func(in LineInput, _ int) api.InvoiceLineReplaceUpdate {
		return api.InvoiceLineReplaceUpdate{
			Name:      in.Name,
			InvoiceAt: in.InvoiceAt,
			Period:    in.Period,
			RateCard:  in.RateCard,
			Price:     in.Price,
			TaxConfig: in.TaxConfig,
		}
	})
}

// Invoice reads an invoice with its lines expanded.
func Invoice(t *testing.T, client *api.ClientWithResponses, invoiceID string) api.Invoice {
	t.Helper()

	expand := []api.InvoiceExpand{api.InvoiceExpandLines}
	resp, err := client.GetInvoiceWithResponse(t.Context(), invoiceID, &api.GetInvoiceParams{Expand: &expand})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "get invoice: %s", string(resp.Body))
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Lines)

	return *resp.JSON200
}

// WithProfileBehavior returns want with the profile's default behavior merged into every tax
// config. Standard invoices merge the invoice's defaultTaxConfig into their lines, so lines
// without a behavior get the profile's. Gathering invoices don't.
func WithProfileBehavior(want map[string]*api.TaxConfig) map[string]*api.TaxConfig {
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

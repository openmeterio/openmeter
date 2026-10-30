package httpdriver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type listAddonsService struct {
	addon.Service

	addons []addon.Addon
	input  addon.ListAddonsInput
}

func (s *listAddonsService) ListAddons(_ context.Context, input addon.ListAddonsInput) (pagination.Result[addon.Addon], error) {
	s.input = input

	return pagination.Result[addon.Addon]{Items: s.addons, TotalCount: len(s.addons)}, nil
}

// countingTaxCodeResolver records BatchResolve calls and serves them from taxCodes.
type countingTaxCodeResolver struct {
	productcatalog.TransactionalTaxCodeResolver

	taxCodes map[string]*taxcode.TaxCode
	calls    [][]string
}

func (r *countingTaxCodeResolver) BatchResolve(_ context.Context, _ string, ids ...string) (map[string]*taxcode.TaxCode, error) {
	r.calls = append(r.calls, ids)

	return lo.PickByKeys(r.taxCodes, ids), nil
}

func TestListAddonsResolvesTaxCodesOnce(t *testing.T) {
	const (
		namespace = "ns"
		taxCodeA  = "01JTAXCODE0000000000000000A"
		taxCodeB  = "01JTAXCODE0000000000000000B"
	)

	stripeCodes := map[string]string{taxCodeA: "txcd_10000000", taxCodeB: "txcd_20030000"}

	resolver := &countingTaxCodeResolver{
		taxCodes: lo.MapValues(stripeCodes, func(code string, id string) *taxcode.TaxCode {
			return &taxcode.TaxCode{
				NamespacedID: models.NamespacedID{Namespace: namespace, ID: id},
				AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: code}},
			}
		}),
	}

	rateCard := func(key, taxCodeID string) productcatalog.RateCard {
		return &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:       key,
				Name:      key,
				TaxConfig: &productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID},
			},
		}
	}

	newAddon := func(id string) addon.Addon {
		return addon.Addon{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: id},
			AddonMeta: productcatalog.AddonMeta{
				Key:          id,
				Name:         id,
				Version:      1,
				Currency:     currencies.NewCurrencyReference("USD"),
				InstanceType: productcatalog.AddonInstanceTypeSingle,
			},
			// taxCodeA is shared by both add-ons, so the batch must deduplicate it.
			RateCards: addon.RateCards{{RateCard: rateCard("a", taxCodeA)}, {RateCard: rateCard("b", taxCodeB)}},
		}
	}

	// given: two add-ons whose rate cards reference two distinct Tax Codes, one of them repeated
	svc := &listAddonsService{addons: []addon.Addon{newAddon("addon-1"), newAddon("addon-2")}}
	handler := httpdriver.New(namespacedriver.StaticNamespaceDecoder(namespace), svc, resolver)

	// when: the add-ons are listed
	rec := httptest.NewRecorder()
	handler.ListAddons().With(api.ListAddonsParams{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// then: the Tax Codes are looked up once, with each ID once
	assert.Equal(t, []string{namespace}, svc.input.Namespaces)
	require.Len(t, resolver.calls, 1)
	assert.ElementsMatch(t, []string{taxCodeA, taxCodeB}, resolver.calls[0])

	// then: every rate card shows its Tax Code ID and the Stripe code mapped to it
	var resp httpdriver.ListAddonsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Items, 2)

	for _, a := range resp.Items {
		require.Len(t, a.RateCards, 2)

		for _, rc := range a.RateCards {
			flat, err := rc.AsRateCardFlatFee()
			require.NoError(t, err)
			require.NotNil(t, flat.TaxConfig, "rate card %s", flat.Key)
			require.NotNil(t, flat.TaxConfig.TaxCodeId, "rate card %s", flat.Key)
			require.NotNil(t, flat.TaxConfig.Stripe, "rate card %s", flat.Key)
			assert.Equal(t, stripeCodes[*flat.TaxConfig.TaxCodeId], flat.TaxConfig.Stripe.Code, "rate card %s", flat.Key)
		}
	}
}

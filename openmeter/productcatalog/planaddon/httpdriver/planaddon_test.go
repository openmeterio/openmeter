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

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/httpdriver"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type listPlanAddonsService struct {
	planaddon.Service

	planAddons []planaddon.PlanAddon
	input      planaddon.ListPlanAddonsInput
}

func (s *listPlanAddonsService) ListPlanAddons(_ context.Context, input planaddon.ListPlanAddonsInput) (pagination.Result[planaddon.PlanAddon], error) {
	s.input = input

	return pagination.Result[planaddon.PlanAddon]{Items: s.planAddons, TotalCount: len(s.planAddons)}, nil
}

func TestListPlanAddonsResolvesTaxCodesOnce(t *testing.T) {
	const (
		namespace = "ns"
		taxCodeA  = "01JTAXC0DE0000000000000000A"
		taxCodeB  = "01JTAXC0DE0000000000000000B"
	)

	stripeCodes := map[string]string{taxCodeA: "txcd_10000000", taxCodeB: "txcd_20030000"}

	resolver := &pctestutils.CountingTaxCodeResolver{
		TaxCodes: lo.MapValues(stripeCodes, func(code string, id string) *taxcode.TaxCode {
			return &taxcode.TaxCode{
				NamespacedID: models.NamespacedID{Namespace: namespace, ID: id},
				AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: code}},
			}
		}),
	}

	original, target := lookupTestPlanAddon(namespace, "addon-1", taxCodeA, taxCodeB), lookupTestPlanAddon(namespace, "addon-2", taxCodeA, taxCodeB)

	// given: two plan add-ons whose add-on rate cards reference two distinct Tax Codes, one of them repeated
	svc := &listPlanAddonsService{planAddons: []planaddon.PlanAddon{original, target}}
	handler := httpdriver.New(namespacedriver.StaticNamespaceDecoder(namespace), svc, resolver)

	// when: the plan's add-ons are listed
	rec := httptest.NewRecorder()
	handler.ListPlanAddons().With(httpdriver.ListPlanAddonsParams{PlanIDOrKey: "plan-1"}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// then: the Tax Codes are looked up once, with each ID once
	assert.Equal(t, []string{namespace}, svc.input.Namespaces)
	require.Len(t, resolver.Calls, 1)
	assert.ElementsMatch(t, []string{taxCodeA, taxCodeB}, resolver.Calls[0])

	// then: every rate card shows its Tax Code ID and the Stripe code mapped to it
	var resp httpdriver.ListPlanAddonsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Items, 2)

	for _, pa := range resp.Items {
		require.Len(t, pa.Addon.RateCards, 2)

		for _, rc := range pa.Addon.RateCards {
			flat, err := rc.AsRateCardFlatFee()
			require.NoError(t, err)
			require.NotNil(t, flat.TaxConfig, "rate card %s", flat.Key)
			require.NotNil(t, flat.TaxConfig.TaxCodeId, "rate card %s", flat.Key)
			require.NotNil(t, flat.TaxConfig.Stripe, "rate card %s", flat.Key)
			assert.Equal(t, stripeCodes[*flat.TaxConfig.TaxCodeId], flat.TaxConfig.Stripe.Code, "rate card %s", flat.Key)
		}
	}
}

// lookupTestRateCard returns a flat fee rate card referencing the given Tax Code by ID.
func lookupTestRateCard(key, taxCodeID string) productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:       key,
			Name:      key,
			TaxConfig: &productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID},
		},
	}
}

// lookupTestPlanAddon returns a minimal plan add-on assignment whose add-on rate cards reference
// two Tax Codes; taxCodeA is shared by both cards, so a listing batch must deduplicate it.
func lookupTestPlanAddon(namespace, key, taxCodeA, taxCodeB string) planaddon.PlanAddon {
	return planaddon.PlanAddon{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: "plan-1-" + key},
		Plan: plan.Plan{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: "plan-1"},
			PlanMeta: productcatalog.PlanMeta{
				Key:      "plan-1",
				Name:     "plan-1",
				Version:  1,
				Currency: currencies.NewCurrencyReference("USD"),
			},
		},
		Addon: lookupTestAddon(namespace, key, taxCodeA, taxCodeB),
	}
}

// lookupTestAddon returns a minimal add-on whose rate cards reference two Tax Codes; taxCodeA is
// shared by both cards, so a listing batch must deduplicate it.
func lookupTestAddon(namespace, key, taxCodeA, taxCodeB string) addon.Addon {
	return addon.Addon{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: key},
		AddonMeta: productcatalog.AddonMeta{
			Key:          key,
			Name:         key,
			Version:      1,
			Currency:     currencies.NewCurrencyReference("USD"),
			InstanceType: productcatalog.AddonInstanceTypeSingle,
		},
		RateCards: addon.RateCards{{RateCard: lookupTestRateCard("a", taxCodeA)}, {RateCard: lookupTestRateCard("b", taxCodeB)}},
	}
}

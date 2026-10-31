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
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type listPlansService struct {
	plan.Service

	plans []plan.Plan
	input plan.ListPlansInput
}

func (s *listPlansService) ListPlans(_ context.Context, input plan.ListPlansInput) (pagination.Result[plan.Plan], error) {
	s.input = input

	return pagination.Result[plan.Plan]{Items: s.plans, TotalCount: len(s.plans)}, nil
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

func TestListPlansResolvesTaxCodesOnce(t *testing.T) {
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

	newPlan := func(id string) plan.Plan {
		return plan.Plan{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: id},
			PlanMeta: productcatalog.PlanMeta{
				Key:      id,
				Name:     id,
				Version:  1,
				Currency: currencies.NewCurrencyReference("USD"),
			},
			Phases: []plan.Phase{{
				Phase: productcatalog.Phase{
					PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
					// taxCodeA is shared by both plans, so the batch must deduplicate it.
					RateCards: productcatalog.RateCards{rateCard("a", taxCodeA), rateCard("b", taxCodeB)},
				},
			}},
		}
	}

	// given: two plans whose rate cards reference two distinct Tax Codes, one of them repeated
	svc := &listPlansService{plans: []plan.Plan{newPlan("plan-1"), newPlan("plan-2")}}
	handler := httpdriver.New(namespacedriver.StaticNamespaceDecoder(namespace), svc, resolver)

	// when: the plans are listed
	rec := httptest.NewRecorder()
	handler.ListPlans().With(api.ListPlansParams{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// then: the Tax Codes are looked up once, with each ID once
	assert.Equal(t, []string{namespace}, svc.input.Namespaces)
	require.Len(t, resolver.calls, 1)
	assert.ElementsMatch(t, []string{taxCodeA, taxCodeB}, resolver.calls[0])

	// then: every rate card shows its Tax Code ID and the Stripe code mapped to it
	var resp httpdriver.ListPlansResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Items, 2)

	for _, p := range resp.Items {
		require.Len(t, p.Phases, 1)
		require.Len(t, p.Phases[0].RateCards, 2)

		for _, rc := range p.Phases[0].RateCards {
			flat, err := rc.AsRateCardFlatFee()
			require.NoError(t, err)
			require.NotNil(t, flat.TaxConfig, "rate card %s", flat.Key)
			require.NotNil(t, flat.TaxConfig.TaxCodeId, "rate card %s", flat.Key)
			require.NotNil(t, flat.TaxConfig.Stripe, "rate card %s", flat.Key)
			assert.Equal(t, stripeCodes[*flat.TaxConfig.TaxCodeId], flat.TaxConfig.Stripe.Code, "rate card %s", flat.Key)
		}
	}
}

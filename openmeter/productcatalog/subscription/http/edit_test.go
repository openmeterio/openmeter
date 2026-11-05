package httpdriver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptionhttp "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/taxcoderesolver"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// editSubscriptionSuite holds the fixtures shared by the v1 edit tests: the handler under test,
// the services behind it, and the subscription being edited.
type editSubscriptionSuite struct {
	handler        subscriptionhttp.Handler
	deps           subscriptiontestutils.SubscriptionDependencies
	ns             string
	phaseKey       string
	subscriptionID string
}

// addItemOp renders an add_item edit operation with a rate card carrying the given stripe.code.
func (s *editSubscriptionSuite) addItemOp(itemKey, stripeCode string) string {
	return fmt.Sprintf(`{
		"op": "add_item",
		"phaseKey": %q,
		"rateCard": {
			"type": "flat_fee",
			"key": %q,
			"name": %q,
			"billingCadence": "P1M",
			"price": {"type": "flat", "amount": "10", "paymentTerm": "in_advance"},
			"taxConfig": {"behavior": "exclusive", "stripe": {"code": %q}}
		}
	}`, s.phaseKey, itemKey, itemKey, stripeCode)
}

// edit serves the edit request with the given customizations.
func (s *editSubscriptionSuite) edit(customizations ...string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	body := fmt.Sprintf(`{"customizations": [%s]}`, strings.Join(customizations, ","))

	s.handler.EditSubscription().
		With(subscriptionhttp.EditSubscriptionParams{ID: s.subscriptionID}).
		ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body)))

	return rec
}

// getTaxCode returns the Tax Code mapped to the Stripe code, if any.
func (s *editSubscriptionSuite) getTaxCode(ctx context.Context, stripeCode string) (taxcode.TaxCode, error) {
	return s.deps.TaxCodeEnv.Service.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
		Namespace: s.ns,
		AppType:   app.AppTypeStripe,
		TaxCode:   stripeCode,
	})
}

func TestV1EditSubscriptionResolvesLegacyStripeTaxCodes(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	cus := deps.CustomerAdapter.CreateExampleCustomer(t)
	ns := cus.Namespace

	taxCodeResolver, err := taxcoderesolver.NewTransactional(deps.TaxCodeEnv.Service, deps.TaxCodeEnv.Adapter)
	require.NoError(t, err)

	handler := subscriptionhttp.NewHandler(subscriptionhttp.HandlerConfig{
		SubscriptionService:         deps.SubscriptionService,
		SubscriptionWorkflowService: deps.WorkflowService,
		CustomerService:             deps.CustomerService,
		NamespaceDecoder:            namespacedriver.StaticNamespaceDecoder(ns),
		Logger:                      testutils.NewLogger(t),
		TaxCodeResolver:             taxCodeResolver,
	})

	view, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: lo.ToPtr(now.Add(-time.Second))},
		},
		Namespace:  ns,
		CustomerID: cus.ID,
	}, &plansubscription.Plan{
		Plan: subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, subscriptiontestutils.ExampleRateCard2.Clone()).Build().Plan,
	})
	require.NoError(t, err)
	require.Len(t, view.Phases, 1)

	phaseKey := view.Phases[0].SubscriptionPhase.Key

	s := editSubscriptionSuite{
		handler:        handler,
		deps:           deps,
		ns:             ns,
		phaseKey:       phaseKey,
		subscriptionID: view.Subscription.ID,
	}

	t.Run("added item references a Tax Code created for its stripe.code", func(t *testing.T) {
		// given: no Tax Code is mapped to the Stripe code yet
		_, err := s.getTaxCode(t.Context(), "txcd_20000000")
		require.True(t, taxcode.IsTaxCodeNotFoundError(err), "expected not found, got: %v", err)

		// when: an item with that legacy stripe.code is added through the v1 API
		rec := s.edit(s.addItemOp("taxed", "txcd_20000000"))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		// then: a Tax Code is created and the item references it
		tc, err := s.getTaxCode(t.Context(), "txcd_20000000")
		require.NoError(t, err)

		updated, err := deps.SubscriptionService.GetView(t.Context(), view.Subscription.NamespacedID)
		require.NoError(t, err)

		items := updated.Phases[0].ItemsByKey["taxed"]
		require.Len(t, items, 1)

		taxConfig := items[0].SubscriptionItem.RateCard.AsMeta().TaxConfig
		require.NotNil(t, taxConfig)
		require.Equal(t, tc.ID, taxConfig.TaxCodeID)

		// and: the v1 read renders the stripe.code from the Tax Code's mapping
		rec = httptest.NewRecorder()
		handler.GetSubscription().
			With(subscriptionhttp.GetSubscriptionParams{ID: view.Subscription.ID}).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var got api.SubscriptionExpanded
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

		item, ok := lo.Find(got.Phases[0].Items, func(item api.SubscriptionItem) bool { return item.Key == "taxed" })
		require.True(t, ok)
		require.NotNil(t, item.TaxConfig)
		require.NotNil(t, item.TaxConfig.Stripe)
		require.Equal(t, "txcd_20000000", item.TaxConfig.Stripe.Code)
		require.Equal(t, tc.ID, lo.FromPtr(item.TaxConfig.TaxCodeId))
	})

	t.Run("failed edit doesn't leave a Tax Code behind", func(t *testing.T) {
		// when: an edit adds an item with a new stripe.code but another operation in it fails
		rec := s.edit(
			s.addItemOp("taxed-2", "txcd_30000000"),
			fmt.Sprintf(`{"op": "remove_item", "phaseKey": %q, "itemKey": "does-not-exist"}`, phaseKey),
		)
		require.NotEqual(t, http.StatusOK, rec.Code, rec.Body.String())

		// then: the Tax Code created for the stripe.code is rolled back with the edit
		_, err := s.getTaxCode(t.Context(), "txcd_30000000")
		require.True(t, taxcode.IsTaxCodeNotFoundError(err), "expected not found, got: %v", err)
	})
}

package httpdriver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	customerhttp "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptionhttp "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/ffx"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestV1SubscriptionReadsExcludeCustomCurrency(t *testing.T) {
	// given: one customer with fiat, custom-default, and mixed-currency inline subscriptions.
	now := time.Now().UTC().Truncate(time.Second)
	clock.FreezeTime(now)
	defer clock.UnFreeze()
	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)
	deps := subscriptiontestutils.NewService(t, dbDeps)
	cus := deps.CustomerAdapter.CreateExampleCustomer(t)
	ns := cus.Namespace
	managedCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(ns, "CREDITS", "Credits", "CR"))
	require.NoError(t, err)

	ctx := ffx.SetAccessOnContext(t.Context(), ffx.AccessConfig{subscription.MultiSubscriptionEnabledFF: true})
	views := make(map[string]subscription.SubscriptionView)
	for _, kind := range []string{"custom", "mixed", "fiat"} {
		card := subscriptiontestutils.ExampleRateCard2.Clone()
		require.NoError(t, card.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
			meta.Key = kind
			if kind == "mixed" {
				meta.Currency = lo.ToPtr(managedCurrency.Reference())
			}
			return meta, nil
		}))
		input := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, card).Build()
		input.SettlementMode = productcatalog.CreditOnlySettlementMode
		if kind == "custom" {
			input.Currency = managedCurrency.Reference()
		}
		if kind == "mixed" {
			input.Phases[0].RateCards = append(input.Phases[0].RateCards, subscriptiontestutils.ExampleRateCard2.Clone())
		}
		view, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{Custom: lo.ToPtr(now.Add(-time.Second))},
			},
			Namespace:  ns,
			CustomerID: cus.ID,
		}, &plansubscription.Plan{Plan: input.Plan})
		require.NoError(t, err)
		require.Equal(t, currencyx.Code("USD"), view.Subscription.InvoiceCurrency)
		require.Nil(t, view.Subscription.PlanRef, "visibility must use subscription items, including inline plans")
		views[kind] = view
	}
	decoder := namespacedriver.StaticNamespaceDecoder(ns)
	handler := subscriptionhttp.NewHandler(subscriptionhttp.HandlerConfig{
		SubscriptionService: deps.SubscriptionService,
		CustomerService:     deps.CustomerService,
		NamespaceDecoder:    decoder,
		Logger:              testutils.NewLogger(t),
	})

	t.Run("get", func(t *testing.T) {
		// when: each subscription is read through V1.
		// then: custom and mixed currencies are rejected while fiat remains available.
		for kind, view := range views {
			t.Run(kind, func(t *testing.T) {
				rec := httptest.NewRecorder()
				handler.GetSubscription().With(subscriptionhttp.GetSubscriptionParams{ID: view.Subscription.ID}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
				if kind == "fiat" {
					require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				} else {
					require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
					require.Contains(t, rec.Body.String(), "currency_not_representable")
				}
			})
		}
	})
	t.Run("paginated list", func(t *testing.T) {
		// when: requesting a page smaller than the unfiltered subscription set.
		// then: filtering precedes pagination and total-count calculation.
		for _, page := range []int{1, 2} {
			rec := httptest.NewRecorder()
			handler.ListCustomerSubscriptions().With(subscriptionhttp.ListCustomerSubscriptionsParams{
				CustomerIDOrKey: cus.ID, Params: api.ListCustomerSubscriptionsParams{Page: &page, PageSize: lo.ToPtr(1)},
			}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var result pagination.Result[api.Subscription]
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
			require.Equal(t, 1, result.TotalCount)
			if page == 1 {
				require.Len(t, result.Items, 1)
				require.Equal(t, views["fiat"].Subscription.ID, result.Items[0].Id)
			} else {
				require.Empty(t, result.Items)
			}
		}
	})
	customerHandler := customerhttp.New(decoder, deps.CustomerService, deps.SubscriptionService, nil)
	t.Run("customer get expansion", func(t *testing.T) {
		// when: reading the customer's expanded subscriptions.
		// then: the customer remains visible with only its fiat subscription.
		rec := httptest.NewRecorder()
		customerHandler.GetCustomer().With(customerhttp.GetCustomerParams{
			CustomerIDOrKey:   cus.ID,
			GetCustomerParams: api.GetCustomerParams{Expand: lo.ToPtr([]api.CustomerExpand{api.CustomerExpandSubscriptions})},
		}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var result api.Customer
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		require.NotNil(t, result.Subscriptions)
		require.Len(t, *result.Subscriptions, 1)
		require.Equal(t, views["fiat"].Subscription.ID, (*result.Subscriptions)[0].Id)
	})
	t.Run("customer list expansion", func(t *testing.T) {
		// when: listing customers with the default subscription expansion.
		// then: custom-currency subscriptions are omitted from the embedded list too.
		rec := httptest.NewRecorder()
		customerHandler.ListCustomers().With(api.ListCustomersParams{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var result pagination.Result[api.Customer]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		require.Len(t, result.Items, 1)
		require.NotNil(t, result.Items[0].Subscriptions)
		require.Len(t, *result.Items[0].Subscriptions, 1)
		require.Equal(t, views["fiat"].Subscription.ID, (*result.Items[0].Subscriptions)[0].Id)
	})
	t.Run("unfiltered service", func(t *testing.T) {
		// when: V3 and internal callers list without the V1 compatibility filter.
		// then: all subscriptions remain available.
		result, err := deps.SubscriptionService.List(t.Context(), subscription.ListSubscriptionsInput{Namespaces: []string{ns}})
		require.NoError(t, err)
		require.Len(t, result.Items, 3)
	})
	t.Run("deleted custom currency revision", func(t *testing.T) {
		// given: a custom-currency subscription whose rate card is replaced with fiat.
		view := views["custom"]
		target := view.Spec
		for _, phase := range target.Phases {
			replacement := *phase.ItemsByKey["custom"][0]
			replacement.ItemKey = "replacement"
			replacement.RateCard = replacement.RateCard.Clone()
			require.NoError(t, replacement.RateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
				meta.Key = replacement.ItemKey
				meta.Currency = lo.ToPtr(currencies.NewCurrencyReference(currencyx.Code("USD")))
				return meta, nil
			}))
			phase.ItemsByKey = map[string][]*subscription.SubscriptionItemSpec{replacement.ItemKey: {&replacement}}
		}
		// when: the service persists the replacement and soft-deletes the old item.
		_, err := deps.SubscriptionService.Update(ctx, view.Subscription.NamespacedID, target)
		require.NoError(t, err)
		oldItem := view.Phases[0].ItemsByKey["custom"][0].SubscriptionItem
		row, err := dbDeps.DBClient.SubscriptionItem.Get(t.Context(), oldItem.ID)
		require.NoError(t, err)
		require.NotNil(t, row.DeletedAt)
		require.NotNil(t, row.CustomCurrencyID)

		// then: the obsolete custom-currency row no longer hides the fiat subscription.
		rec := httptest.NewRecorder()
		handler.GetSubscription().With(subscriptionhttp.GetSubscriptionParams{ID: view.Subscription.ID}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = httptest.NewRecorder()
		handler.ListCustomerSubscriptions().With(subscriptionhttp.ListCustomerSubscriptionsParams{CustomerIDOrKey: cus.ID}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var result pagination.Result[api.Subscription]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		require.ElementsMatch(t, []string{views["fiat"].Subscription.ID, view.Subscription.ID}, lo.Map(result.Items, func(item api.Subscription, _ int) string { return item.Id }))
	})
}

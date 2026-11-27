package subscriptions

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/featureresolver"
	pcservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestSubscriptionEntitlementConflict(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// Given a customer with a running subscription and a separately granted feature.
	db := subscriptiontestutils.SetupDBDeps(t)
	defer db.Cleanup(t)
	deps := subscriptiontestutils.NewService(t, db)
	feature := deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)[0]
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, subscriptiontestutils.ExampleRateCard2.Clone()).Build())
	sub := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, plan, start)
	existing, err := deps.EntitlementRegistry.Entitlement.ScheduleEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        sub.Customer.Namespace,
		UsageAttribution: sub.Customer.GetUsageAttribution(),
		FeatureID:        &feature.ID,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	})
	require.NoError(t, err)

	target := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, subscriptiontestutils.ExampleRateCard4ForAddons.Clone()).Build()
	target.Key = "with-feature"
	deps.PlanHelper.CreatePlan(t, target)

	clock.FreezeTime(start.Add(24 * time.Hour))
	defer clock.UnFreeze()

	resolver, err := featureresolver.New(deps.FeatureConnector)
	require.NoError(t, err)
	service, err := pcservice.New(pcservice.Config{
		WorkflowService: deps.WorkflowService, SubscriptionService: deps.SubscriptionService,
		PlanService: deps.PlanService, FeatureResolver: resolver, CurrencyResolver: deps.CurrencyResolver,
		CustomerService: deps.CustomerService, TaxCodeResolver: deps.TaxCodeResolver, Logger: testutils.NewLogger(t),
	})
	require.NoError(t, err)
	h := &handler{
		resolveNamespace:            func(context.Context) (string, error) { return sub.Customer.Namespace, nil },
		subscriptionWorkflowService: deps.WorkflowService,
		subscriptionService:         deps.SubscriptionService,
		planService:                 deps.PlanService,
		planSubscriptionService:     service,
	}

	for _, tc := range []struct {
		name    string
		handler http.Handler
		body    string
	}{
		{
			name:    "edit",
			handler: h.EditSubscription().With(sub.Subscription.ID),
			body: fmt.Sprintf(`{"customizations":[{
				"type":"add_item", "phase_key":%q,
				"rate_card":{"key":%q,"name":"Extra","price":{"type":"free"},"feature":{"id":%q},"entitlement":{"type":"boolean"}}
			}]}`, sub.Phases[0].SubscriptionPhase.Key, feature.Key, feature.ID),
		},
		{
			name:    "change",
			handler: h.ChangeSubscription().With(sub.Subscription.ID),
			body:    fmt.Sprintf(`{"plan":{"key":%q},"timing":"immediate"}`, target.Key),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// When a subscription operation would grant the same feature.
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/"+tc.name, strings.NewReader(tc.body))
			w := httptest.NewRecorder()
			tc.handler.ServeHTTP(w, req)

			// Then the response identifies the conflicting entitlement.
			require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), existing.ID)
		})
	}
}

package subscriptions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/featureresolver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	pcservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFromAPIBillingSubscriptionMigrate(t *testing.T) {
	id := models.NamespacedID{Namespace: "test", ID: "subscription"}

	t.Run("omitted fields retain domain defaults and allow unit config", func(t *testing.T) {
		got, err := FromAPIBillingSubscriptionMigrate(id, api.BillingSubscriptionMigrate{})
		require.NoError(t, err)
		require.Equal(t, plansubscription.MigrateSubscriptionRequest{ID: id}, got)
	})

	t.Run("maps version, phase, anchor and next cycle", func(t *testing.T) {
		anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		var timing api.BillingSubscriptionEditTiming
		require.NoError(t, timing.FromBillingSubscriptionEditTimingEnum(api.BillingSubscriptionEditTimingEnumNextBillingCycle))

		got, err := FromAPIBillingSubscriptionMigrate(id, api.BillingSubscriptionMigrate{
			TargetVersion: lo.ToPtr(2), StartingPhase: lo.ToPtr("paid"), BillingAnchor: &anchor, Timing: &timing,
		})
		require.NoError(t, err)
		require.Equal(t, plansubscription.MigrateSubscriptionRequest{
			ID: id, TargetVersion: lo.ToPtr(2), StartingPhase: lo.ToPtr("paid"), BillingAnchor: &anchor,
			Timing: &subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)},
		}, got)
	})

	t.Run("maps custom timing", func(t *testing.T) {
		at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		var timing api.BillingSubscriptionEditTiming
		require.NoError(t, timing.FromDateTime(at))

		got, err := FromAPIBillingSubscriptionMigrate(id, api.BillingSubscriptionMigrate{Timing: &timing})
		require.NoError(t, err)
		require.Equal(t, &subscription.Timing{Custom: &at}, got.Timing)
	})
}

func TestMigrateSubscriptionHandler(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a running subscription and a later plan that adds an item
	db := subscriptiontestutils.SetupDBDeps(t)
	defer db.Cleanup(t)
	deps := subscriptiontestutils.NewService(t, db)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, subscriptiontestutils.ExampleRateCard1.Clone()).Build())
	before := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, plan, start)

	clock.FreezeTime(start.Add(24 * time.Hour))
	deps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, subscriptiontestutils.ExampleRateCard1.Clone(), subscriptiontestutils.ExampleRateCard2.Clone()).Build())
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))

	resolver, err := featureresolver.New(deps.FeatureConnector)
	require.NoError(t, err)
	service, err := pcservice.New(pcservice.Config{
		WorkflowService: deps.WorkflowService, SubscriptionService: deps.SubscriptionService,
		PlanService: deps.PlanService, FeatureResolver: resolver, CurrencyResolver: deps.CurrencyResolver,
		CustomerService: deps.CustomerService, Logger: testutils.NewLogger(t),
	})
	require.NoError(t, err)

	h := &handler{
		resolveNamespace:        func(context.Context) (string, error) { return before.Subscription.Namespace, nil },
		planSubscriptionService: service,
	}

	// when the HTTP handler migrates in place
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/migrate", strings.NewReader(`{}`))
	writer := httptest.NewRecorder()
	h.MigrateSubscription().With(before.Subscription.ID).ServeHTTP(writer, req)
	require.Equal(t, http.StatusOK, writer.Code, writer.Body.String())

	// then current remains the original snapshot and next contains the amended view
	var result api.BillingSubscriptionMigrateResponse
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &result))
	require.Equal(t, before.Subscription.ID, result.Current.Id)
	require.Equal(t, result.Current.Id, result.Next.Id)
	require.Equal(t, 1, result.Current.Plan.Version)
	require.Equal(t, 2, result.Next.Plan.Version)
	var wire struct {
		Current map[string]json.RawMessage `json:"current"`
	}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &wire))
	require.NotContains(t, wire.Current, "phases")
	require.NotContains(t, wire.Current, "current_period")
	require.Len(t, result.Next.Phases, 1)
	require.Len(t, result.Next.Phases[0].Items, 2)

	t.Run("same version is a bad request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/migrate", strings.NewReader(`{"target_version":2}`))
		writer := httptest.NewRecorder()
		h.MigrateSubscription().With(before.Subscription.ID).ServeHTTP(writer, req)
		require.Equal(t, http.StatusBadRequest, writer.Code, writer.Body.String())
	})

	t.Run("missing subscription is not found", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/migrate", strings.NewReader(`{}`))
		writer := httptest.NewRecorder()
		h.MigrateSubscription().With("01K00000000000000000000000").ServeHTTP(writer, req)
		require.Equal(t, http.StatusNotFound, writer.Code, writer.Body.String())
	})
}

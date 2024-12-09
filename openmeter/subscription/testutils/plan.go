package subscriptiontestutils

import (
	"context"
	"log/slog"
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionplan "github.com/openmeterio/openmeter/openmeter/subscription/adapters/plan"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func GetExamplePlanInput(t *testing.T) plan.CreatePlanInput {
	return plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: ExampleNamespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:     "Test Plan",
				Key:      "test_plan",
				Version:  1,
				Currency: currency.USD,
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "test_phase_1",
						Name:        "Test Phase 1",
						Description: lo.ToPtr("Test Phase 1 Description"),
						StartAfter:  testutils.GetISODuration(t, "P0M"),
					},
					RateCards: productcatalog.RateCards{
						&ExampleRateCard1,
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "test_phase_2",
						Name:        "Test Phase 2",
						Description: lo.ToPtr("Test Phase 2 Description"),
						StartAfter:  testutils.GetISODuration(t, "P1M"),
					},
					RateCards: productcatalog.RateCards{
						&ExampleRateCard1,
						&ExampleRateCard2,
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "test_phase_3",
						Name:        "Test Phase 3",
						Description: lo.ToPtr("Test Phase 3 Description"),
						StartAfter:  testutils.GetISODuration(t, "P3M"),
					},
					RateCards: productcatalog.RateCards{
						&ExampleRateCard1,
					},
				},
			},
		},
	}
}

type planAdapter struct {
	subscription.PlanAdapter
	planService plan.Service
}

func NewPlanAdapter(t *testing.T, dbDeps *DBDeps, logger *slog.Logger, featureConnector feature.FeatureConnector) *planAdapter {
	t.Helper()

	planRepo, err := planrepo.New(planrepo.Config{
		Client: dbDeps.dbClient,
		Logger: logger,
	})

	require.Nil(t, err)

	planService, err := planservice.New(planservice.Config{
		Feature: featureConnector,
		Adapter: planRepo,
		Logger:  testutils.NewLogger(t),
	})

	require.Nil(t, err)

	return &planAdapter{
		planService: planService,
		PlanAdapter: subscriptionplan.NewSubscriptionPlanAdapter(
			subscriptionplan.PlanSubscriptionAdapterConfig{
				PlanService: planService,
				Logger:      logger,
			},
		),
	}
}

func (a *planAdapter) CreateExamplePlan(t *testing.T, ctx context.Context) subscription.Plan {
	t.Helper()

	p, err := a.planService.CreatePlan(ctx, GetExamplePlanInput(t))
	require.Nil(t, err)
	require.NotNil(t, p)

	p, err = a.planService.PublishPlan(ctx, plan.PublishPlanInput{
		NamespacedID: p.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(clock.Now()),
			EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2030-01-01T00:00:00Z")),
		},
	})

	require.Nil(t, err)
	require.NotNil(t, p)

	return &subscriptionplan.SubscriptionPlan{
		Plan: *p,
	}
}

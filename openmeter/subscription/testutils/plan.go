package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
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

// PlanHelper simply creates and returns a plan
type planHelper struct {
	planService plan.Service
}

func NewPlanHelper(planService plan.Service) *planHelper {
	return &planHelper{
		planService: planService,
	}
}

func (h *planHelper) CreatePlan(t *testing.T, input plan.CreatePlanInput) subscription.Plan {
	t.Helper()
	ctx := context.Background()

	p, err := h.planService.CreatePlan(ctx, GetExamplePlanInput(t))
	require.Nil(t, err)
	require.NotNil(t, p)

	p, err = h.planService.PublishPlan(ctx, plan.PublishPlanInput{
		NamespacedID: p.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(clock.Now()),
			EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2030-01-01T00:00:00Z")),
		},
	})

	require.Nil(t, err)
	require.NotNil(t, p)

	pp, err := p.AsProductCatalogPlan(clock.Now())
	require.Nil(t, err)

	return &plansubscription.Plan{
		Plan: pp,
	}
}

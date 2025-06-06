package subscriptiontestutils

import (
	"context"
	"fmt"
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
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type testPlanbuilder struct {
	p plan.CreatePlanInput
}

func (b *testPlanbuilder) AddPhase(dur *isodate.Period, rcs ...productcatalog.RateCard) *testPlanbuilder {
	idx := len(b.p.Plan.Phases) + 1

	b.p.Plan.Phases = append(b.p.Plan.Phases, productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{
			Key:         fmt.Sprintf("test_phase_%d", idx),
			Name:        fmt.Sprintf("Test Phase %d", idx),
			Description: lo.ToPtr(fmt.Sprintf("Test Phase %d Description", idx)),
			Duration:    dur,
		},
		RateCards: rcs,
	})

	return b
}

func (b *testPlanbuilder) SetMeta(meta productcatalog.PlanMeta) *testPlanbuilder {
	b.p.Plan.PlanMeta = meta
	return b
}

func (b *testPlanbuilder) Build() plan.CreatePlanInput {
	return b.p
}

func BuildTestPlan(t *testing.T) *testPlanbuilder {
	b := &testPlanbuilder{
		p: plan.CreatePlanInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: ExampleNamespace,
			},
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Name:           "Test Plan",
					Key:            "test_plan",
					Version:        1,
					Currency:       currency.USD,
					BillingCadence: isodate.MustParse(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
				Phases: []productcatalog.Phase{},
			},
		},
	}

	return b
}

func GetExamplePlanInput(t *testing.T) plan.CreatePlanInput {
	b := BuildTestPlan(t)

	b.AddPhase(lo.ToPtr(testutils.GetISODuration(t, "P1M")), ExampleRateCard1.Clone())
	b.AddPhase(lo.ToPtr(testutils.GetISODuration(t, "P2M")), ExampleRateCard1.Clone(), ExampleRateCard2.Clone(), ExampleRateCard3ForAddons.Clone())
	b.AddPhase(nil, ExampleRateCard1.Clone(), ExampleRateCard3ForAddons.Clone())

	return b.Build()
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

	p, err := h.planService.CreatePlan(ctx, input)
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

	require.Nilf(t, p.DeletedAt, "plan %s should not be deleted", p.NamespacedID.ID)

	return &plansubscription.Plan{
		Plan: p.AsProductCatalogPlan(),
		Ref:  &p.NamespacedID,
	}
}

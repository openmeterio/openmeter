package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

type examplePlanOptions struct {
	taxConfig *productcatalog.TaxConfig
}

type ExamplePlanOption func(*examplePlanOptions)

// WithTaxConfig replaces the shared example tax config, for tests that assert on their own tax
// code, the legacy Stripe-only input, or in-memory tests without a tax code env.
func WithTaxConfig(taxConfig productcatalog.TaxConfig) ExamplePlanOption {
	return func(o *examplePlanOptions) {
		o.taxConfig = &taxConfig
	}
}

// GetExamplePlanInput builds the example plan in ExampleNamespace. Its taxed rate cards use the
// shared example tax code (ExampleTaxConfig) unless WithTaxConfig is given; env may be nil then.
func GetExamplePlanInput(t *testing.T, env *taxcodetestutils.TestEnv, opts ...ExamplePlanOption) plan.CreatePlanInput {
	t.Helper()

	var o examplePlanOptions
	for _, opt := range opts {
		opt(&o)
	}

	taxConfig := lo.FromPtr(o.taxConfig)
	if o.taxConfig == nil {
		taxConfig = ExampleTaxConfig(t, env)
	}

	b := BuildTestPlanInput(t)

	b.AddPhase(lo.ToPtr(datetime.MustParseDuration(t, "P1M")), ExampleRateCard1(taxConfig))
	b.AddPhase(lo.ToPtr(datetime.MustParseDuration(t, "P2M")), ExampleRateCard1(taxConfig), ExampleRateCard2.Clone(), ExampleRateCard3ForAddons(taxConfig))
	b.AddPhase(nil, ExampleRateCard1(taxConfig), ExampleRateCard3ForAddons(taxConfig))

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

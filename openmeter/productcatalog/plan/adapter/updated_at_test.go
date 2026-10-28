package adapter_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestListPlansUpdatedAt(t *testing.T) {
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := pctestutils.NewTestNamespace(t)

	createPlan := func(key string) *plan.Plan {
		// Rate card tax config is irrelevant to the plan row's update time and would
		// need a provisioned tax code.
		input := pctestutils.NewTestPlan(t, namespace, func(_ *testing.T, p *productcatalog.Plan) {
			p.Key = key
			for _, phase := range p.Phases {
				for _, rateCard := range phase.RateCards {
					if flatFee, ok := rateCard.(*productcatalog.FlatFeeRateCard); ok {
						flatFee.TaxConfig = nil
					}
				}
			}
		})

		created, err := env.PlanRepository.CreatePlan(t.Context(), input)
		require.NoError(t, err)

		return created
	}

	list := func(updatedAt filter.FilterTime) pagination.Result[plan.Plan] {
		result, err := env.PlanRepository.ListPlans(t.Context(), plan.ListPlansInput{
			Namespaces: []string{namespace},
			Page:       pagination.NewPage(1, 1),
			UpdatedAt:  &updatedAt,
		})
		require.NoError(t, err)

		return result
	}

	ids := func(result pagination.Result[plan.Plan]) []string {
		return lo.Map(result.Items, func(p plan.Plan, _ int) string { return p.ID })
	}

	// given: a plan written before a sync checkpoint and one written at it.
	checkpoint := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	clock.FreezeTime(checkpoint.Add(-time.Hour))
	defer clock.UnFreeze()
	before := createPlan("before-checkpoint")

	clock.FreezeTime(checkpoint)
	atCheckpoint := createPlan("at-checkpoint")

	// when/then: gte includes the boundary, gt excludes it, and lt selects the
	// earlier plan; page totals count only the filtered rows.
	gte := list(filter.FilterTime{Gte: &checkpoint})
	require.Equal(t, []string{atCheckpoint.ID}, ids(gte))
	require.Equal(t, 1, gte.TotalCount)

	require.Empty(t, list(filter.FilterTime{Gt: &checkpoint}).Items)

	lt := list(filter.FilterTime{Lt: &checkpoint})
	require.Equal(t, []string{before.ID}, ids(lt))
	require.Equal(t, 1, lt.TotalCount)
}

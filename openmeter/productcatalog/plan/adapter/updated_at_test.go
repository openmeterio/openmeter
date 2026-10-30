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

	// given: two plans written before a sync checkpoint.
	checkpoint := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	clock.FreezeTime(checkpoint.Add(-time.Hour))
	defer clock.UnFreeze()
	createPlan("unchanged")
	changed := createPlan("changed")

	// when: one of them is updated at the checkpoint.
	clock.FreezeTime(checkpoint)
	defer clock.UnFreeze()

	_, err := env.PlanRepository.UpdatePlan(t.Context(), plan.UpdatePlanInput{
		NamespacedID: changed.NamespacedID,
		Name:         lo.ToPtr("changed-renamed"),
	})
	require.NoError(t, err)

	// then: only the updated plan matches. Its created_at is before the checkpoint,
	// so this can only match on updated_at.
	result, err := env.PlanRepository.ListPlans(t.Context(), plan.ListPlansInput{
		Namespaces: []string{namespace},
		Page:       pagination.NewPage(1, 10),
		UpdatedAt:  &filter.FilterTime{Gte: &checkpoint},
	})
	require.NoError(t, err)
	require.Equal(t, []string{changed.ID}, lo.Map(result.Items, func(p plan.Plan, _ int) string { return p.ID }))
}

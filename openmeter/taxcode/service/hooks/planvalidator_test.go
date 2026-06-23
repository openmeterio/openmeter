package hooks_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/taxcode/service/hooks"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// stubPlanService implements plan.Service for testing the plan validator hook.
// Only ListPlans has real behavior; all other methods panic.
type stubPlanService struct {
	listResult pagination.Result[plan.Plan]
	listErr    error
	lastInput  plan.ListPlansInput
}

func (s *stubPlanService) ListPlans(ctx context.Context, params plan.ListPlansInput) (pagination.Result[plan.Plan], error) {
	s.lastInput = params
	return s.listResult, s.listErr
}

func (s *stubPlanService) CreatePlan(_ context.Context, _ plan.CreatePlanInput) (*plan.Plan, error) {
	panic("not implemented")
}

func (s *stubPlanService) DeletePlan(_ context.Context, _ plan.DeletePlanInput) error {
	panic("not implemented")
}

func (s *stubPlanService) GetPlan(_ context.Context, _ plan.GetPlanInput) (*plan.Plan, error) {
	panic("not implemented")
}

func (s *stubPlanService) UpdatePlan(_ context.Context, _ plan.UpdatePlanInput) (*plan.Plan, error) {
	panic("not implemented")
}

func (s *stubPlanService) PublishPlan(_ context.Context, _ plan.PublishPlanInput) (*plan.Plan, error) {
	panic("not implemented")
}

func (s *stubPlanService) ArchivePlan(_ context.Context, _ plan.ArchivePlanInput) (*plan.Plan, error) {
	panic("not implemented")
}

func (s *stubPlanService) NextPlan(_ context.Context, _ plan.NextPlanInput) (*plan.Plan, error) {
	panic("not implemented")
}

var _ plan.Service = (*stubPlanService)(nil)

const testTaxCodeID = "01234567890123456789012345"

func TestPlanValidatorHook_PreDelete(t *testing.T) {
	tc := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{
			Namespace: "test-ns",
			ID:        testTaxCodeID,
		},
	}

	t.Run("blocks deletion when a plan references the tax code", func(t *testing.T) {
		// given: a plan service that returns one matching plan
		stub := &stubPlanService{
			listResult: pagination.Result[plan.Plan]{
				Items: []plan.Plan{
					{ManagedModel: models.ManagedModel{}, NamespacedID: models.NamespacedID{ID: "plan-abc"}},
				},
				TotalCount: 1,
			},
		}

		hook, err := hooks.NewPlanValidatorHook(hooks.PlanValidatorHookConfig{PlanService: stub})
		require.NoError(t, err)

		// when: PreDelete is called
		err = hook.PreDelete(t.Context(), tc)

		// then: an error is returned and it is a TaxCodeReferencedByPlan error
		require.Error(t, err)
		require.True(t, taxcode.IsTaxCodeReferencedByPlanError(err),
			"expected TaxCodeReferencedByPlan error, got: %v", err)

		// and: the stub received a ListPlansInput whose TaxCodes.In contains the tax code id
		require.NotNil(t, stub.lastInput.TaxCodes)
		require.NotNil(t, stub.lastInput.TaxCodes.In)
		require.Contains(t, *stub.lastInput.TaxCodes.In, testTaxCodeID)
	})

	t.Run("allows deletion when no plan references the tax code", func(t *testing.T) {
		// given: a plan service that returns no matching plans
		stub := &stubPlanService{
			listResult: pagination.Result[plan.Plan]{
				Items:      []plan.Plan{},
				TotalCount: 0,
			},
		}

		hook, err := hooks.NewPlanValidatorHook(hooks.PlanValidatorHookConfig{PlanService: stub})
		require.NoError(t, err)

		// when: PreDelete is called
		err = hook.PreDelete(t.Context(), tc)

		// then: no error is returned
		require.NoError(t, err)
	})
}

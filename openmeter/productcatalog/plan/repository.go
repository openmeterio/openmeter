package plan

import (
	"context"

	planentity "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// TODO: add bulk api

type Repository interface {
	entutils.TxCreator
	// Plans

	ListPlans(ctx context.Context, params ListPlansInput) (pagination.PagedResponse[planentity.Plan], error)
	CreatePlan(ctx context.Context, params CreatePlanInput) (*planentity.Plan, error)
	DeletePlan(ctx context.Context, params DeletePlanInput) error
	GetPlan(ctx context.Context, params GetPlanInput) (*planentity.Plan, error)
	UpdatePlan(ctx context.Context, params UpdatePlanInput) (*planentity.Plan, error)

	// Phases

	ListPhases(ctx context.Context, params ListPhasesInput) (pagination.PagedResponse[planentity.Phase], error)
	CreatePhase(ctx context.Context, params CreatePhaseInput) (*planentity.Phase, error)
	DeletePhase(ctx context.Context, params DeletePhaseInput) error
	GetPhase(ctx context.Context, params GetPhaseInput) (*planentity.Phase, error)
	UpdatePhase(ctx context.Context, params UpdatePhaseInput) (*planentity.Phase, error)
}

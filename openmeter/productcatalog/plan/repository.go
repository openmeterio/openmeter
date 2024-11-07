package plan

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

// TODO: add bulk api

type Repository interface {
	// Plans

	ListPlans(ctx context.Context, params ListPlansInput) (pagination.PagedResponse[Plan], error)
	CreatePlan(ctx context.Context, params CreatePlanInput) (*Plan, error)
	DeletePlan(ctx context.Context, params DeletePlanInput) error
	GetPlan(ctx context.Context, params GetPlanInput) (*Plan, error)
	UpdatePlan(ctx context.Context, params UpdatePlanInput) (*Plan, error)

	// Phases

	ListPhases(ctx context.Context, params ListPhasesInput) (pagination.PagedResponse[Phase], error)
	CreatePhase(ctx context.Context, params CreatePhaseInput) (*Phase, error)
	DeletePhase(ctx context.Context, params DeletePhaseInput) error
	GetPhase(ctx context.Context, params GetPhaseInput) (*Phase, error)
	UpdatePhase(ctx context.Context, params UpdatePhaseInput) (*Phase, error)
}

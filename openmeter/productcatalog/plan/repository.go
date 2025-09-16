package plan

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// TODO: add bulk api

type Repository interface {
	entutils.TxCreator
	// Plans

	ListPlans(ctx context.Context, params ListPlansInput) (pagination.Result[Plan], error)
	CreatePlan(ctx context.Context, params CreatePlanInput) (*Plan, error)
	DeletePlan(ctx context.Context, params DeletePlanInput) error
	GetPlan(ctx context.Context, params GetPlanInput) (*Plan, error)
	UpdatePlan(ctx context.Context, params UpdatePlanInput) (*Plan, error)
}

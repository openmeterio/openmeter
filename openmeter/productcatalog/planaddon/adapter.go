package planaddon

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	entutils.TxCreator

	ListPlanAddons(ctx context.Context, params ListPlanAddonsInput) (pagination.Result[PlanAddon], error)
	CreatePlanAddon(ctx context.Context, params CreatePlanAddonInput) (*PlanAddon, error)
	DeletePlanAddon(ctx context.Context, params DeletePlanAddonInput) error
	GetPlanAddon(ctx context.Context, params GetPlanAddonInput) (*PlanAddon, error)
	UpdatePlanAddon(ctx context.Context, params UpdatePlanAddonInput) (*PlanAddon, error)
}

// Repository is an alias for Adapter for backward compatibility.
type Repository = Adapter

package progressmanager

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
)

type Adapter interface {
	ProgressManagerAdapter
}

type ProgressManagerAdapter interface {
	GetProgress(ctx context.Context, input entity.GetProgressInput) (*entity.Progress, error)
	UpsertProgress(ctx context.Context, input entity.UpsertProgressInput) error
}

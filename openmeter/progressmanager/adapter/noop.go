package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	"github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
)

func (a *adapterNoop) GetProgress(ctx context.Context, input entity.GetProgressInput) (*entity.Progress, error) {
	return nil, progressmanager.NotFoundError{ID: input.ID, Entity: "progress"}
}

func (a *adapterNoop) UpsertProgress(ctx context.Context, input entity.UpsertProgressInput) error {
	return nil
}

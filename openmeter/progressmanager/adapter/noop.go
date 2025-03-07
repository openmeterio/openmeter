package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapterNoop) GetProgress(ctx context.Context, input entity.GetProgressInput) (*entity.Progress, error) {
	return nil, models.NewGenericNotFoundError(
		fmt.Errorf("progress not found for id: %s", input.ProgressID.ID),
	)
}

func (a *adapterNoop) UpsertProgress(ctx context.Context, input entity.UpsertProgressInput) error {
	return nil
}

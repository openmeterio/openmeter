package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerdimensiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerdimension"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (r *repo) CreateDimension(ctx context.Context, input ledgeraccount.CreateDimensionInput) (*ledgeraccount.DimensionData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.DimensionData, error) {
		entity, err := r.db.LedgerDimension.Create().
			SetNamespace(input.Namespace).
			SetAnnotations(input.Annotations).
			SetDimensionKey(input.Key).
			SetDimensionValue(input.Value).
			SetDimensionDisplayValue(input.DisplayValue).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create ledger dimension: %w", err)
		}

		return MapDimensionData(entity)
	})
}

func (r *repo) GetDimensionByID(ctx context.Context, id models.NamespacedID) (*ledgeraccount.DimensionData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.DimensionData, error) {
		entity, err := r.db.LedgerDimension.Query().
			Where(
				ledgerdimensiondb.Namespace(id.Namespace),
				ledgerdimensiondb.ID(id.ID),
			).
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get ledger dimension by id: %w", err)
		}

		return MapDimensionData(entity)
	})
}

func MapDimensionData(entity *db.LedgerDimension) (*ledgeraccount.DimensionData, error) {
	dKey := ledger.DimensionKey(entity.DimensionKey)
	if err := dKey.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dimension key: %w", err)
	}

	return &ledgeraccount.DimensionData{
		ID:          entity.ID,
		Namespace:   entity.Namespace,
		CreatedAt:   entity.CreatedAt,
		Annotations: entity.Annotations,
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		DimensionKey:          dKey,
		DimensionValue:        entity.DimensionValue,
		DimensionDisplayValue: entity.DimensionDisplayValue,
	}, nil
}

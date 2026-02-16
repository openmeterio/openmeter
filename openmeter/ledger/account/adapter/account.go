package adapter

import (
	"context"
	"fmt"

	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	ledgerdimensiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerdimension"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (r *repo) CreateAccount(ctx context.Context, input ledgeraccount.CreateAccountInput) (ledgeraccount.Account, error) {
	entity, err := r.db.LedgerAccount.Create().
		SetNamespace(input.Namespace).
		SetAccountType(input.Type).
		SetAnnotations(input.Annotations).
		Save(ctx)
	if err != nil {
		return ledgeraccount.Account{}, fmt.Errorf("failed to create ledger account: %w", err)
	}

	acc := ledgeraccount.NewAccountFromData(nil, ledgeraccount.AccountData{
		ID: models.NamespacedID{
			Namespace: entity.Namespace,
			ID:        entity.ID,
		},
		Annotations: input.Annotations,
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		AccountType: entity.AccountType,
	})

	return *acc, nil
}

func (r *repo) GetAccountByID(ctx context.Context, id models.NamespacedID) (ledgeraccount.Account, error) {
	entity, err := r.db.LedgerAccount.Query().
		Where(
			ledgeraccountdb.Namespace(id.Namespace),
			ledgeraccountdb.ID(id.ID),
		).
		Only(ctx)
	if err != nil {
		return ledgeraccount.Account{}, fmt.Errorf("failed to get ledger account by id: %w", err)
	}

	acc := ledgeraccount.NewAccountFromData(nil, ledgeraccount.AccountData{
		ID: models.NamespacedID{
			Namespace: entity.Namespace,
			ID:        entity.ID,
		},
		Annotations: entity.Annotations,
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		AccountType: entity.AccountType,
	})

	return *acc, nil
}

func (r *repo) CreateDimension(ctx context.Context, input ledgeraccount.CreateDimensionInput) (ledgeraccount.Dimension, error) {
	entity, err := r.db.LedgerDimension.Create().
		SetNamespace(input.Namespace).
		SetAnnotations(input.Annotations).
		SetDimensionKey(input.Key).
		SetDimensionValue(input.Value).
		Save(ctx)
	if err != nil {
		return ledgeraccount.Dimension{}, fmt.Errorf("failed to create ledger dimension: %w", err)
	}

	return ledgeraccount.Dimension{
		ID: models.NamespacedID{
			Namespace: entity.Namespace,
			ID:        entity.ID,
		},
		Annotations: entity.Annotations,
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		DimensionKey:   entity.DimensionKey,
		DimensionValue: entity.DimensionValue,
	}, nil
}

func (r *repo) GetDimensionByID(ctx context.Context, id models.NamespacedID) (ledgeraccount.Dimension, error) {
	entity, err := r.db.LedgerDimension.Query().
		Where(
			ledgerdimensiondb.Namespace(id.Namespace),
			ledgerdimensiondb.ID(id.ID),
		).
		Only(ctx)
	if err != nil {
		return ledgeraccount.Dimension{}, fmt.Errorf("failed to get ledger dimension by id: %w", err)
	}

	return ledgeraccount.Dimension{
		ID: models.NamespacedID{
			Namespace: entity.Namespace,
			ID:        entity.ID,
		},
		Annotations: entity.Annotations,
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		DimensionKey:   entity.DimensionKey,
		DimensionValue: entity.DimensionValue,
	}, nil
}

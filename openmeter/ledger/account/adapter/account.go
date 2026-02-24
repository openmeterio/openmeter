package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (r *repo) CreateAccount(ctx context.Context, input ledgeraccount.CreateAccountInput) (*ledgeraccount.AccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.AccountData, error) {
		entity, err := r.db.LedgerAccount.Create().
			SetNamespace(input.Namespace).
			SetAccountType(input.Type).
			SetAnnotations(input.Annotations).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create ledger account: %w", err)
		}

		return MapAccountData(entity)
	})
}

func (r *repo) GetAccountByID(ctx context.Context, id models.NamespacedID) (*ledgeraccount.AccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.AccountData, error) {
		entity, err := r.db.LedgerAccount.Query().
			Where(
				ledgeraccountdb.Namespace(id.Namespace),
				ledgeraccountdb.ID(id.ID),
			).
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get ledger account by id: %w", err)
		}

		return MapAccountData(entity)
	})
}

func (r *repo) ListAccounts(ctx context.Context, input ledgeraccount.ListAccountsInput) ([]*ledgeraccount.AccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) ([]*ledgeraccount.AccountData, error) {
		q := r.db.LedgerAccount.Query().
			Where(ledgeraccountdb.Namespace(input.Namespace))

		if len(input.AccountTypes) > 0 {
			q = q.Where(ledgeraccountdb.AccountTypeIn(input.AccountTypes...))
		}

		entities, err := q.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list ledger accounts: %w", err)
		}

		out := make([]*ledgeraccount.AccountData, 0, len(entities))
		for _, entity := range entities {
			accData, err := MapAccountData(entity)
			if err != nil {
				return nil, fmt.Errorf("failed to map account data: %w", err)
			}
			out = append(out, accData)
		}

		return out, nil
	})
}

func MapAccountData(entity *db.LedgerAccount) (*ledgeraccount.AccountData, error) {
	return &ledgeraccount.AccountData{
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
	}, nil
}

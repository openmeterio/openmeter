package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbledgersubaccount "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (r *repo) CreateSubAccount(ctx context.Context, input ledgeraccount.CreateSubAccountInput) (*ledgeraccount.SubAccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.SubAccountData, error) {
		entity, err := r.db.LedgerSubAccount.Create().
			SetNamespace(input.Namespace).
			SetAnnotations(input.Annotations).
			SetAccountID(input.AccountID).
			SetCurrencyDimensionID(input.Dimensions.CurrencyDimensionID).
			SetNillableTaxCodeDimensionID(input.Dimensions.TaxCodeDimensionID).
			SetNillableFeaturesDimensionID(input.Dimensions.FeaturesDimensionID).
			SetNillableCreditPriorityDimensionID(input.Dimensions.CreditPriorityDimensionID).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create ledger sub-account: %w", err)
		}

		// We need to load the edges
		res, err := r.GetSubAccountByID(ctx, models.NamespacedID{
			Namespace: input.Namespace,
			ID:        entity.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get ledger sub-account: %w", err)
		}

		return res, nil
	})
}

func (r *repo) GetSubAccountByID(ctx context.Context, id models.NamespacedID) (*ledgeraccount.SubAccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.SubAccountData, error) {
		entity, err := r.db.LedgerSubAccount.Query().
			Where(dbledgersubaccount.ID(id.ID)).
			Where(dbledgersubaccount.Namespace(id.Namespace)).
			WithCurrencyDimension().
			WithTaxCodeDimension().
			WithFeaturesDimension().
			WithCreditPriorityDimension().
			WithAccount().
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get ledger sub-account: %w", err)
		}

		subAccountData, err := MapSubAccountData(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map sub-account data: %w", err)
		}

		return &subAccountData, nil
	})
}

func MapSubAccountData(entity *db.LedgerSubAccount) (ledgeraccount.SubAccountData, error) {
	if entity.Edges.Account == nil {
		return ledgeraccount.SubAccountData{}, fmt.Errorf("account edge is required")
	}

	dimensions := ledgeraccount.SubAccountDimensions{}

	{
		if entity.Edges.CurrencyDimension == nil {
			return ledgeraccount.SubAccountData{}, fmt.Errorf("currency dimension edge is required")
		}

		currencyDimensionData, err := MapDimensionData(entity.Edges.CurrencyDimension)
		if err != nil {
			return ledgeraccount.SubAccountData{}, fmt.Errorf("failed to map currency dimension data: %w", err)
		}

		cDim, err := currencyDimensionData.AsCurrencyDimension()
		if err != nil {
			return ledgeraccount.SubAccountData{}, fmt.Errorf("failed to map currency dimension: %w", err)
		}

		dimensions.Currency = cDim
	}

	return ledgeraccount.SubAccountData{
		ID:          entity.ID,
		Namespace:   entity.Namespace,
		Annotations: entity.Annotations,
		CreatedAt:   entity.CreatedAt,
		AccountID:   entity.AccountID,
		AccountType: entity.Edges.Account.AccountType,
		Dimensions:  dimensions,
	}, nil
}

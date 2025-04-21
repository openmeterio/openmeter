package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/ledger"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingsubledger"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ ledger.SubledgerAdapter = (*adapter)(nil)

func (a *adapter) UpsertSubledger(ctx context.Context, input ledger.UpsertSubledgerAdapterInput) (ledger.Subledger, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (ledger.Subledger, error) {
		subledger, err := tx.db.BillingSubledger.Query().
			Where(billingsubledger.Namespace(input.LedgerID.Namespace), billingsubledger.LedgerID(input.LedgerID.ID), billingsubledger.Key(input.Key)).
			First(ctx)
		if err != nil {
			if !entdb.IsNotFound(err) {
				return ledger.Subledger{}, fmt.Errorf("failed to get subledger: %w", err)
			}
			subledger, err = tx.db.BillingSubledger.Create().
				SetNamespace(input.LedgerID.Namespace).
				SetKey(input.Key).
				SetPriority(input.Priority).
				SetName(input.Name).
				SetNillableDescription(input.Description).
				SetLedgerID(input.LedgerID.ID).
				Save(ctx)
		}
		return mapSubledgerFromDB(subledger), nil
	})
}

func mapSubledgerFromDB(dbSubledger *entdb.BillingSubledger) ledger.Subledger {
	return ledger.Subledger{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:          dbSubledger.ID,
			Namespace:   dbSubledger.Namespace,
			CreatedAt:   dbSubledger.CreatedAt,
			UpdatedAt:   dbSubledger.UpdatedAt,
			DeletedAt:   dbSubledger.DeletedAt,
			Name:        dbSubledger.Name,
			Description: dbSubledger.Description,
		}),
		Key:      dbSubledger.Key,
		LedgerID: dbSubledger.LedgerID,
		Priority: dbSubledger.Priority,
	}
}

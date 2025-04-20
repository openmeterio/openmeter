package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/ledger"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) CreateTransaction(ctx context.Context, input ledger.CreateTransactionInput) (ledger.Transaction, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (ledger.Transaction, error) {
		var ownerType, ownerID string
		if input.Owner != nil {
			ownerType = string(input.Owner.Type)
			ownerID = input.Owner.ID
		}

		trns, err := tx.db.BillingSubledgerTransaction.Create().
			// Resource
			SetNamespace(input.Subledger.Namespace).
			SetName(input.Name).
			SetNillableDescription(input.Description).
			SetMetadata(input.Metadata).

			// Native fields
			SetLedgerID(input.Subledger.LedgerID).
			SetSubledgerID(input.Subledger.ID).
			SetAmount(input.Amount).
			SetOwnerType(ownerType).
			SetOwnerID(ownerID).
			Save(ctx)
		if err != nil {
			return ledger.Transaction{}, err
		}

		return mapTransactionFromDB(trns), nil
	})
}

func mapTransactionFromDB(dbTrns *entdb.BillingSubledgerTransaction) ledger.Transaction {
	var ownerType *ledger.OwnerReference
	if dbTrns.OwnerType != nil && dbTrns.OwnerID != nil {
		ownerType = &ledger.OwnerReference{Type: ledger.OwnerReferenceType(*dbTrns.OwnerType), ID: *dbTrns.OwnerID}
	}

	return ledger.Transaction{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:        dbTrns.ID,
			Namespace: dbTrns.Namespace,
			CreatedAt: dbTrns.CreatedAt,
			UpdatedAt: dbTrns.UpdatedAt,
			DeletedAt: dbTrns.DeletedAt,
			Name:      dbTrns.Name,
		}),
		SubledgerID: dbTrns.SubledgerID,
		LedgerID:    dbTrns.LedgerID,
		Amount:      dbTrns.Amount,
		Owner:       ownerType,
		Metadata:    dbTrns.Metadata,
	}
}

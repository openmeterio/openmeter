package postgres_connector

import (
	"context"
	"time"

	credit_model "github.com/openmeterio/openmeter/internal/credit"
	db_credit "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
)

func (a *PostgresConnector) GetLedger(
	ctx context.Context,
	namespace string,
	subject string,
	from time.Time,
	to time.Time,
) (credit_model.LedgerEntryList, error) {
	ledgerEntries := credit_model.NewLedgerEntryList()

	entities, err := a.db.CreditEntry.Query().Where(
		db_credit.And(
			db_credit.EntryTypeEQ(credit_model.EntryTypeReset),
			db_credit.EffectiveAtGTE(from),
			db_credit.EffectiveAtLTE(to),
		),
	).All(ctx)
	if err != nil {
		return ledgerEntries, err
	}

	resets := []time.Time{}
	for _, entity := range entities {
		reset, err := mapResetEntity(entity)
		if err != nil {
			return ledgerEntries, err
		}

		ledgerEntries.AddReset(reset)
		resets = append(resets, reset.EffectiveAt)
	}
	resets = append(resets, to)

	balanceFrom := from
	for _, balanceTo := range resets {
		_, entries, err := a.getBalance(ctx, namespace, subject, balanceFrom, balanceTo)
		if err != nil {
			return ledgerEntries, err
		}
		ledgerEntries.Append(entries)
		balanceFrom = balanceTo
	}

	return ledgerEntries, nil
}

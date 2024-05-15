package postgres_connector

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	db_credit "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
)

func (a *PostgresConnector) GetHistory(
	ctx context.Context,
	ledgerID credit.NamespacedID,
	from time.Time,
	to time.Time,
	limit int,
) (credit.LedgerEntryList, error) {
	ledgerEntries := credit.NewLedgerEntryList()

	query := a.db.CreditEntry.Query().Where(
		db_credit.And(
			db_credit.EntryTypeEQ(credit.EntryTypeReset),
			db_credit.EffectiveAtGTE(from),
			db_credit.EffectiveAtLTE(to),
		),
	).Order(
		db_credit.ByCreatedAt(),
	)

	if limit > 0 {
		query = query.Limit(limit)
	}

	entities, err := query.All(ctx)
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
		_, entries, err := a.getBalance(ctx, ledgerID, balanceFrom, balanceTo)
		if err != nil {
			return ledgerEntries, err
		}
		ledgerEntries.Append(entries)
		balanceFrom = balanceTo
	}

	return ledgerEntries, nil
}

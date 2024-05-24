// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package postgres_connector

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	db_credit "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
)

func (a *PostgresConnector) GetHistory(
	ctx context.Context,
	ledgerID credit.NamespacedLedgerID,
	from time.Time,
	to time.Time,
	pagination credit.Pagination,
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

	// Because of the above we cannot really limit the query from the db side,
	// so we are "emulating" the limit here
	if pagination.Offset > 0 {
		ledgerEntries = ledgerEntries.Skip(pagination.Offset)
	}

	if pagination.Limit > 0 && ledgerEntries.Len() > pagination.Limit {
		ledgerEntries = ledgerEntries.Truncate(pagination.Limit)
	}

	return ledgerEntries, nil
}

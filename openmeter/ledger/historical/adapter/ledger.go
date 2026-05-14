package adapter

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"slices"

	sql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgersubaccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	ledgertransactiongroupdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransactiongroup"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func hydrateHistoricalTransaction(tx *db.LedgerTransaction) (*ledgerhistorical.Transaction, error) {
	entryData, err := slicesx.MapWithErr(tx.Edges.Entries, func(entry *db.LedgerEntry) (ledgerhistorical.EntryData, error) {
		subAccount, err := entry.Edges.SubAccountOrErr()
		if err != nil {
			return ledgerhistorical.EntryData{}, fmt.Errorf("entry %s missing sub-account edge: %w", entry.ID, err)
		}

		account, err := subAccount.Edges.AccountOrErr()
		if err != nil {
			return ledgerhistorical.EntryData{}, fmt.Errorf("entry %s sub-account %s missing account edge: %w", entry.ID, subAccount.ID, err)
		}
		route, err := subAccount.Edges.RouteOrErr()
		if err != nil {
			return ledgerhistorical.EntryData{}, fmt.Errorf("entry %s sub-account %s missing route edge: %w", entry.ID, subAccount.ID, err)
		}

		return ledgerhistorical.EntryData{
			ID:           entry.ID,
			Namespace:    entry.Namespace,
			Annotations:  entry.Annotations,
			CreatedAt:    entry.CreatedAt,
			IdentityKey:  entry.IdentityKey,
			SubAccountID: entry.SubAccountID,
			AccountType:  account.AccountType,
			Route: ledger.Route{
				Currency:                       currencyx.Code(route.Currency),
				TaxCode:                        route.TaxCode,
				Features:                       route.Features,
				CostBasis:                      route.CostBasis,
				CreditPriority:                 route.CreditPriority,
				TransactionAuthorizationStatus: route.TransactionAuthorizationStatus,
			},
			RouteID:       route.ID,
			RouteKey:      route.RoutingKey,
			RouteKeyVer:   route.RoutingKeyVersion,
			Amount:        entry.Amount,
			TransactionID: entry.TransactionID,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("transaction %s entry hydration failed: %w", tx.ID, err)
	}

	reconstructed, err := ledgerhistorical.NewTransactionFromData(
		ledgerhistorical.TransactionData{
			ID:          tx.ID,
			Namespace:   tx.Namespace,
			Annotations: tx.Annotations,
			CreatedAt:   tx.CreatedAt,
			BookedAt:    tx.BookedAt,
		},
		entryData,
	)
	if err != nil {
		return nil, fmt.Errorf("transaction %s: %w", tx.ID, err)
	}

	return reconstructed, nil
}

func (r *repo) BookTransaction(ctx context.Context, groupID models.NamespacedID, input ledger.TransactionInput) (*ledgerhistorical.Transaction, error) {
	if input == nil {
		return nil, ledger.ErrTransactionInputRequired
	}

	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgerhistorical.Transaction, error) {
		entity, err := tx.db.LedgerTransaction.Create().
			SetNamespace(groupID.Namespace).
			SetGroupID(groupID.ID).
			SetAnnotations(input.Annotations()).
			SetBookedAt(input.BookedAt()).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create ledger transaction: %w", err)
		}

		entryInputs := input.EntryInputs()
		accountTypesBySubAccountID := make(map[string]ledger.AccountType, len(entryInputs))
		routeIDBySubAccountID := make(map[string]string, len(entryInputs))
		routeKeyBySubAccountID := make(map[string]string, len(entryInputs))
		routeKeyVersionBySubAccountID := make(map[string]ledger.RoutingKeyVersion, len(entryInputs))
		routeBySubAccountID := make(map[string]ledger.Route, len(entryInputs))
		createInputs := make([]*db.LedgerEntryCreate, 0, len(entryInputs))
		for _, entryInput := range entryInputs {
			subAccountID := entryInput.PostingAddress().SubAccountID()
			route := entryInput.PostingAddress().Route()
			accountTypesBySubAccountID[subAccountID] = entryInput.PostingAddress().AccountType()
			routeIDBySubAccountID[subAccountID] = route.ID()
			routeKeyBySubAccountID[subAccountID] = route.RoutingKey().Value()
			routeKeyVersionBySubAccountID[subAccountID] = route.RoutingKey().Version()
			routeBySubAccountID[subAccountID] = route.Route()

			createInputs = append(createInputs, tx.db.LedgerEntry.Create().
				SetNamespace(groupID.Namespace).
				SetSubAccountID(subAccountID).
				SetIdentityKey(entryInput.IdentityKey()).
				SetAnnotations(entryInput.Annotations()).
				SetAmount(entryInput.Amount()).
				SetTransactionID(entity.ID))
		}

		createdEntries := make([]*db.LedgerEntry, 0, len(createInputs))
		if len(createInputs) > 0 {
			createdEntries, err = tx.db.LedgerEntry.CreateBulk(createInputs...).Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create ledger entries: %w", err)
			}
		}

		transaction, err := ledgerhistorical.NewTransactionFromData(
			ledgerhistorical.TransactionData{
				ID:          entity.ID,
				Namespace:   entity.Namespace,
				Annotations: entity.Annotations,
				CreatedAt:   entity.CreatedAt,
				BookedAt:    entity.BookedAt,
			},
			lo.Map(createdEntries, func(e *db.LedgerEntry, _ int) ledgerhistorical.EntryData {
				return ledgerhistorical.EntryData{
					ID:            e.ID,
					Namespace:     e.Namespace,
					Annotations:   e.Annotations,
					CreatedAt:     e.CreatedAt,
					IdentityKey:   e.IdentityKey,
					SubAccountID:  e.SubAccountID,
					AccountType:   accountTypesBySubAccountID[e.SubAccountID],
					Route:         routeBySubAccountID[e.SubAccountID],
					RouteID:       routeIDBySubAccountID[e.SubAccountID],
					RouteKey:      routeKeyBySubAccountID[e.SubAccountID],
					RouteKeyVer:   routeKeyVersionBySubAccountID[e.SubAccountID],
					Amount:        e.Amount,
					TransactionID: e.TransactionID,
				}
			}),
		)
		if err != nil {
			return nil, fmt.Errorf("ledger transaction view: %w", err)
		}

		return transaction, nil
	})
}

func (r *repo) CreateTransactionGroup(ctx context.Context, transactionGroup ledgerhistorical.CreateTransactionGroupInput) (ledgerhistorical.TransactionGroupData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (ledgerhistorical.TransactionGroupData, error) {
		entity, err := tx.db.LedgerTransactionGroup.Create().
			SetNamespace(transactionGroup.Namespace).
			SetAnnotations(transactionGroup.Annotations).
			Save(ctx)
		if err != nil {
			return ledgerhistorical.TransactionGroupData{}, fmt.Errorf("failed to create transaction group: %w", err)
		}

		return ledgerhistorical.TransactionGroupData{
			ID:          entity.ID,
			Namespace:   entity.Namespace,
			CreatedAt:   entity.CreatedAt,
			Annotations: entity.Annotations,
		}, nil
	})
}

func (r *repo) GetTransactionGroup(ctx context.Context, id models.NamespacedID) (*ledgerhistorical.TransactionGroup, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgerhistorical.TransactionGroup, error) {
		entity, err := tx.db.LedgerTransactionGroup.Query().
			Where(
				ledgertransactiongroupdb.Namespace(id.Namespace),
				ledgertransactiongroupdb.ID(id.ID),
			).
			WithTransactions(func(q *db.LedgerTransactionQuery) {
				q.Order(
					ledgertransactiondb.ByCreatedAt(),
					ledgertransactiondb.ByID(),
				)
				q.WithEntries(func(eq *db.LedgerEntryQuery) {
					eq.Order(
						ledgerentrydb.ByCreatedAt(),
						ledgerentrydb.ByID(),
					)
					eq.WithSubAccount(func(sq *db.LedgerSubAccountQuery) {
						sq.WithAccount()
						sq.WithRoute()
					})
				})
			}).
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query transaction group: %w", err)
		}

		transactions, err := slicesx.MapWithErr(entity.Edges.Transactions, hydrateHistoricalTransaction)
		if err != nil {
			return nil, fmt.Errorf("failed to hydrate transaction group transactions: %w", err)
		}

		return ledgerhistorical.NewTransactionGroupFromData(
			ledgerhistorical.TransactionGroupData{
				ID:          entity.ID,
				Namespace:   entity.Namespace,
				CreatedAt:   entity.CreatedAt,
				Annotations: entity.Annotations,
			},
			transactions,
		), nil
	})
}

func (r *repo) SumEntries(ctx context.Context, query ledger.Query) (alpacadecimal.Decimal, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (alpacadecimal.Decimal, error) {
		q := sumEntriesQuery{
			query: query,
		}

		entryQuery, err := q.Build(tx.db)
		if err != nil {
			return alpacadecimal.Decimal{}, err
		}

		var rows []struct {
			SumAmount stdsql.NullString `json:"sum_amount,omitempty"`
		}

		err = entryQuery.
			Aggregate(db.As(db.Sum(ledgerentrydb.FieldAmount), "sum_amount")).
			Scan(ctx, &rows)
		if err != nil {
			return alpacadecimal.Decimal{}, fmt.Errorf("failed to query ledger entries to sum: %w", err)
		}

		if len(rows) == 0 || !rows[0].SumAmount.Valid {
			return alpacadecimal.NewFromInt(0), nil
		}

		total, err := alpacadecimal.NewFromString(rows[0].SumAmount.String)
		if err != nil {
			return alpacadecimal.Decimal{}, fmt.Errorf("failed to parse summed amount %q: %w", rows[0].SumAmount.String, err)
		}

		return total, nil
	})
}

func (r *repo) ListTransactions(ctx context.Context, input ledger.ListTransactionsInput) (ledger.ListTransactionsResult, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (ledger.ListTransactionsResult, error) {
		entryPredicates := listTransactionsEntryPredicates(input.AccountIDs, input.Currency)

		query := tx.db.LedgerTransaction.Query().
			Where(ledgertransactiondb.Namespace(input.Namespace)).
			WithEntries(func(q *db.LedgerEntryQuery) {
				if len(entryPredicates) > 0 {
					q.Where(entryPredicates...)
				}
				q.Order(
					ledgerentrydb.ByCreatedAt(),
					ledgerentrydb.ByID(),
				)
				q.WithSubAccount(func(sq *db.LedgerSubAccountQuery) {
					sq.WithAccount()
					sq.WithRoute()
				})
			})

		if input.TransactionID != nil {
			query = query.Where(ledgertransactiondb.ID(input.TransactionID.ID))
		}

		// Scope to specific accounts.
		if len(input.AccountIDs) > 0 {
			query = query.Where(
				ledgertransactiondb.HasEntriesWith(
					ledgerentrydb.HasSubAccountWith(
						ledgersubaccountdb.AccountIDIn(input.AccountIDs...),
					),
				),
			)
		}

		if input.Currency != nil {
			query = query.Where(
				ledgertransactiondb.HasEntriesWith(
					ledgerentrydb.HasSubAccountWith(
						ledgersubaccountdb.HasRouteWith(
							ledgersubaccountroutedb.Currency(string(*input.Currency)),
						),
					),
				),
			)
		}

		// Filter by annotation key-value matches.
		for key, value := range input.AnnotationFilters {
			query = query.Where(func(s *sql.Selector) {
				s.Where(sqljson.ValueEQ(ledgertransactiondb.FieldAnnotations, value, sqljson.Path(key)))
			})
		}

		// Exclude transactions by annotation key-value matches.
		for key, value := range input.ExcludeAnnotationFilters {
			query = query.Where(ledgertransactiondb.Not(func(s *sql.Selector) {
				s.Where(sqljson.ValueEQ(ledgertransactiondb.FieldAnnotations, value, sqljson.Path(key)))
			}))
		}

		if input.CreditMovement != ledger.ListTransactionsCreditMovementUnspecified {
			pred, err := ledgerTransactionCreditMovementPredicate(input.AccountIDs, input.Currency, input.CreditMovement)
			if err != nil {
				return ledger.ListTransactionsResult{}, err
			}
			if pred != nil {
				query = query.Where(pred)
			}
		}

		if input.Cursor != nil {
			query = query.Where(ledgerTransactionAfterCursorPredicate(*input.Cursor))
		}

		if input.Before != nil {
			query = query.Where(ledgerTransactionBeforeCursorPredicate(*input.Before))
		}

		query = query.Order(listTransactionsOrdering(input.Before != nil)...)

		query = query.Limit(input.Limit + 1)

		dbItems, err := query.All(ctx)
		if err != nil {
			return ledger.ListTransactionsResult{}, fmt.Errorf("failed to list transactions: %w", err)
		}
		if len(dbItems) == 0 {
			return ledger.ListTransactionsResult{
				Items: []ledger.Transaction{},
			}, nil
		}

		hasMore := len(dbItems) > input.Limit
		if hasMore {
			dbItems = dbItems[:input.Limit]
		}

		items, err := slicesx.MapWithErr(dbItems, func(tx *db.LedgerTransaction) (*ledgerhistorical.Transaction, error) {
			return hydrateHistoricalTransaction(tx)
		})
		if err != nil {
			return ledger.ListTransactionsResult{}, fmt.Errorf("failed to hydrate listed transactions: %w", err)
		}

		if input.Before != nil {
			slices.Reverse(items)
		}

		var nextCursor *ledger.TransactionCursor
		if hasMore && len(items) > 0 {
			nextItem := items[len(items)-1]
			cursor := nextItem.Cursor()
			nextCursor = &cursor
		}

		return ledger.ListTransactionsResult{
			Items: lo.Map(items, func(item *ledgerhistorical.Transaction, _ int) ledger.Transaction {
				return item
			}),
			NextCursor: nextCursor,
		}, nil
	})
}

func listTransactionsEntryPredicates(accountIDs []string, currency *currencyx.Code) []predicate.LedgerEntry {
	entryPredicates := make([]predicate.LedgerEntry, 0, 2)
	subAccountPredicates := make([]predicate.LedgerSubAccount, 0, 2)

	if len(accountIDs) > 0 {
		subAccountPredicates = append(subAccountPredicates, ledgersubaccountdb.AccountIDIn(accountIDs...))
	}

	if currency != nil {
		subAccountPredicates = append(subAccountPredicates,
			ledgersubaccountdb.HasRouteWith(
				ledgersubaccountroutedb.Currency(string(*currency)),
			),
		)
	}

	if len(subAccountPredicates) > 0 {
		entryPredicates = append(entryPredicates, ledgerentrydb.HasSubAccountWith(subAccountPredicates...))
	}

	return entryPredicates
}

func ledgerTransactionAfterCursorPredicate(cursor ledger.TransactionCursor) predicate.LedgerTransaction {
	return func(s *sql.Selector) {
		s.Where(sql.Or(
			sql.LT(s.C(ledgertransactiondb.FieldBookedAt), cursor.BookedAt),
			sql.And(
				sql.EQ(s.C(ledgertransactiondb.FieldBookedAt), cursor.BookedAt),
				sql.Or(
					sql.LT(s.C(ledgertransactiondb.FieldCreatedAt), cursor.CreatedAt),
					sql.And(
						sql.EQ(s.C(ledgertransactiondb.FieldCreatedAt), cursor.CreatedAt),
						sql.LT(s.C(ledgertransactiondb.FieldID), cursor.ID.ID),
					),
				),
			),
		))
	}
}

func ledgerTransactionBeforeCursorPredicate(cursor ledger.TransactionCursor) predicate.LedgerTransaction {
	return func(s *sql.Selector) {
		s.Where(sql.Or(
			sql.GT(s.C(ledgertransactiondb.FieldBookedAt), cursor.BookedAt),
			sql.And(
				sql.EQ(s.C(ledgertransactiondb.FieldBookedAt), cursor.BookedAt),
				sql.Or(
					sql.GT(s.C(ledgertransactiondb.FieldCreatedAt), cursor.CreatedAt),
					sql.And(
						sql.EQ(s.C(ledgertransactiondb.FieldCreatedAt), cursor.CreatedAt),
						sql.GT(s.C(ledgertransactiondb.FieldID), cursor.ID.ID),
					),
				),
			),
		))
	}
}

func listTransactionsOrdering(before bool) []ledgertransactiondb.OrderOption {
	order := sql.OrderDesc()
	if before {
		order = sql.OrderAsc()
	}

	return []ledgertransactiondb.OrderOption{
		ledgertransactiondb.ByBookedAt(order),
		ledgertransactiondb.ByCreatedAt(order),
		ledgertransactiondb.ByID(order),
	}
}

func ledgerTransactionCreditMovementPredicate(
	accountIDs []string,
	currency *currencyx.Code,
	movement ledger.ListTransactionsCreditMovement,
) (predicate.LedgerTransaction, error) {
	var having *sql.Predicate

	switch movement {
	case ledger.ListTransactionsCreditMovementPositive:
		having = scopedEntryAmountSumPredicate(">")
	case ledger.ListTransactionsCreditMovementNegative:
		having = scopedEntryAmountSumPredicate("<")
	case ledger.ListTransactionsCreditMovementUnspecified:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported credit movement filter: %d", movement)
	}

	return func(s *sql.Selector) {
		s.Where(sql.In(
			s.C(ledgertransactiondb.FieldID),
			scopedFBOMovementTransactionSelector(accountIDs, currency, having),
		))
	}, nil
}

func scopedFBOMovementTransactionSelector(
	accountIDs []string,
	currency *currencyx.Code,
	having *sql.Predicate,
) *sql.Selector {
	entries := sql.Table(ledgerentrydb.Table)
	subAccounts := sql.Table(ledgersubaccountdb.Table)
	accounts := sql.Table(ledgeraccountdb.Table)

	selector := sql.Select(entries.C(ledgerentrydb.FieldTransactionID)).
		From(entries).
		Join(subAccounts).
		On(entries.C(ledgerentrydb.FieldSubAccountID), subAccounts.C(ledgersubaccountdb.FieldID)).
		Join(accounts).
		On(subAccounts.C(ledgersubaccountdb.FieldAccountID), accounts.C(ledgeraccountdb.FieldID)).
		Where(sql.EQ(accounts.C(ledgeraccountdb.FieldAccountType), ledger.AccountTypeCustomerFBO))

	if len(accountIDs) > 0 {
		selector.Where(sql.In(subAccounts.C(ledgersubaccountdb.FieldAccountID), stringsToAny(accountIDs)...))
	}

	if currency != nil {
		routes := sql.Table(ledgersubaccountroutedb.Table)
		selector.
			Join(routes).
			On(subAccounts.C(ledgersubaccountdb.FieldRouteID), routes.C(ledgersubaccountroutedb.FieldID)).
			Where(sql.EQ(routes.C(ledgersubaccountroutedb.FieldCurrency), string(*currency)))
	}

	return selector.
		GroupBy(entries.C(ledgerentrydb.FieldTransactionID)).
		Having(having)
}

func scopedEntryAmountSumPredicate(op string) *sql.Predicate {
	return sql.ExprP(fmt.Sprintf("SUM(%s) %s 0", ledgerentrydb.FieldAmount, op))
}

func stringsToAny(values []string) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}

	return out
}

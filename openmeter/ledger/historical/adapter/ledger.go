package adapter

import (
	"context"
	stdsql "database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
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
	"github.com/openmeterio/openmeter/pkg/models"
	pagepagination "github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
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

	entity, err := r.db.LedgerTransaction.Create().
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

		createInputs = append(createInputs, r.db.LedgerEntry.Create().
			SetNamespace(groupID.Namespace).
			SetSubAccountID(subAccountID).
			SetAmount(entryInput.Amount()).
			SetTransactionID(entity.ID))
	}

	createdEntries := make([]*db.LedgerEntry, 0, len(createInputs))
	if len(createInputs) > 0 {
		createdEntries, err = r.db.LedgerEntry.CreateBulk(createInputs...).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create ledger entries: %w", err)
		}
	}

	tx, err := ledgerhistorical.NewTransactionFromData(
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

	return tx, nil
}

func (r *repo) CreateTransactionGroup(ctx context.Context, transactionGroup ledgerhistorical.CreateTransactionGroupInput) (ledgerhistorical.TransactionGroupData, error) {
	entity, err := r.db.LedgerTransactionGroup.Create().
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
}

func (r *repo) GetTransactionGroup(ctx context.Context, id models.NamespacedID) (*ledgerhistorical.TransactionGroup, error) {
	entity, err := r.db.LedgerTransactionGroup.Query().
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
}

func (r *repo) SumEntries(ctx context.Context, query ledger.Query) (alpacadecimal.Decimal, error) {
	q := sumEntriesQuery{
		query: query,
	}

	entryQuery, err := q.Build(r.db)
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
}

func (r *repo) ListTransactions(ctx context.Context, input ledger.ListTransactionsInput) (pagination.Result[*ledgerhistorical.Transaction], error) {
	query := r.db.LedgerTransaction.Query().
		Where(ledgertransactiondb.Namespace(input.Namespace)).
		WithEntries(func(q *db.LedgerEntryQuery) {
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

	// Filter by annotation key-value matches.
	for key, value := range input.AnnotationFilters {
		query = query.Where(func(s *entsql.Selector) {
			s.Where(sqljson.ValueEQ(ledgertransactiondb.FieldAnnotations, value, sqljson.Path(key)))
		})
	}

	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}

	paged, err := query.Cursor(ctx, input.Cursor)
	if err != nil {
		return pagination.Result[*ledgerhistorical.Transaction]{}, fmt.Errorf("failed to list transactions: %w", err)
	}
	if len(paged.Items) == 0 {
		return pagination.Result[*ledgerhistorical.Transaction]{
			Items: []*ledgerhistorical.Transaction{},
		}, nil
	}

	items, err := slicesx.MapWithErr(paged.Items, func(tx *db.LedgerTransaction) (*ledgerhistorical.Transaction, error) {
		return hydrateHistoricalTransaction(tx)
	})
	if err != nil {
		return pagination.Result[*ledgerhistorical.Transaction]{}, fmt.Errorf("failed to hydrate listed transactions: %w", err)
	}

	return pagination.Result[*ledgerhistorical.Transaction]{
		Items:      items,
		NextCursor: paged.NextCursor,
	}, nil
}

func (r *repo) ListTransactionsByPage(ctx context.Context, input ledger.ListTransactionsByPageInput) (pagepagination.Result[*ledgerhistorical.Transaction], error) {
	entryPredicates := listTransactionsEntryPredicates(input)

	query := r.db.LedgerTransaction.Query().
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
		}).
		Order(
			ledgertransactiondb.ByBookedAt(entsql.OrderDesc()),
			ledgertransactiondb.ByCreatedAt(entsql.OrderDesc()),
			ledgertransactiondb.ByID(entsql.OrderDesc()),
		)

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

	for key, value := range input.AnnotationFilters {
		query = query.Where(func(s *entsql.Selector) {
			s.Where(sqljson.ValueEQ(ledgertransactiondb.FieldAnnotations, value, sqljson.Path(key)))
		})
	}

	if input.CreditMovement != ledger.ListTransactionsCreditMovementUnspecified {
		pred, err := ledgerTransactionCreditMovementPredicate(input.AccountIDs, input.Currency, input.CreditMovement)
		if err != nil {
			return pagepagination.Result[*ledgerhistorical.Transaction]{}, err
		}
		if pred != nil {
			query = query.Where(pred)
		}
	}

	paged, err := query.Paginate(ctx, input.Page)
	if err != nil {
		return pagepagination.Result[*ledgerhistorical.Transaction]{}, fmt.Errorf("list transactions by page: %w", err)
	}

	items, err := slicesx.MapWithErr(paged.Items, func(tx *db.LedgerTransaction) (*ledgerhistorical.Transaction, error) {
		return hydrateHistoricalTransaction(tx)
	})
	if err != nil {
		return pagepagination.Result[*ledgerhistorical.Transaction]{}, fmt.Errorf("hydrate listed transactions: %w", err)
	}

	return pagepagination.Result[*ledgerhistorical.Transaction]{
		Page:       paged.Page,
		TotalCount: paged.TotalCount,
		Items:      items,
	}, nil
}

func listTransactionsEntryPredicates(input ledger.ListTransactionsByPageInput) []predicate.LedgerEntry {
	entryPredicates := make([]predicate.LedgerEntry, 0, 2)
	subAccountPredicates := make([]predicate.LedgerSubAccount, 0, 2)

	if len(input.AccountIDs) > 0 {
		subAccountPredicates = append(subAccountPredicates, ledgersubaccountdb.AccountIDIn(input.AccountIDs...))
	}

	if input.Currency != nil {
		subAccountPredicates = append(subAccountPredicates,
			ledgersubaccountdb.HasRouteWith(
				ledgersubaccountroutedb.Currency(string(*input.Currency)),
			),
		)
	}

	if len(subAccountPredicates) > 0 {
		entryPredicates = append(entryPredicates, ledgerentrydb.HasSubAccountWith(subAccountPredicates...))
	}

	return entryPredicates
}

func ledgerTransactionCreditMovementPredicate(
	accountIDs []string,
	currency *currencyx.Code,
	movement ledger.ListTransactionsCreditMovement,
) (predicate.LedgerTransaction, error) {
	subAccountPredicates := []predicate.LedgerSubAccount{
		ledgersubaccountdb.HasAccountWith(
			ledgeraccountdb.AccountType(ledger.AccountTypeCustomerFBO),
		),
	}

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

	entryPredicates := []predicate.LedgerEntry{
		ledgerentrydb.HasSubAccountWith(subAccountPredicates...),
	}

	switch movement {
	case ledger.ListTransactionsCreditMovementPositive:
		entryPredicates = append(entryPredicates, ledgerentrydb.AmountGT(alpacadecimal.Zero))
	case ledger.ListTransactionsCreditMovementNegative:
		entryPredicates = append(entryPredicates, ledgerentrydb.AmountLT(alpacadecimal.Zero))
	case ledger.ListTransactionsCreditMovementUnspecified:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported credit movement filter: %d", movement)
	}

	return ledgertransactiondb.HasEntriesWith(entryPredicates...), nil
}

package adapter

import (
	"fmt"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgersubaccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/internal/routequery"
	"github.com/openmeterio/openmeter/pkg/models"
)

type sumEntriesQuery struct {
	query ledger.Query
}

func (b *sumEntriesQuery) Build(client *db.Client) (*db.LedgerEntryQuery, error) {
	entryPredicates, err := b.entryPredicates()
	if err != nil {
		return nil, err
	}

	return client.LedgerEntry.Query().Where(entryPredicates...), nil
}

// SQL returns the final SQL shape and args used for sum aggregation.
func (b *sumEntriesQuery) SQL() (string, []any, error) {
	selector, err := b.selector()
	if err != nil {
		return "", nil, err
	}
	query, args := selector.Query()
	return query, args, nil
}

func (b *sumEntriesQuery) selector() (*sql.Selector, error) {
	e := sql.Table(ledgerentrydb.Table)
	selector := sql.Select(sql.As(sql.Sum(e.C(ledgerentrydb.FieldAmount)), "sum_amount")).From(e)
	selector.SetDialect(dialect.Postgres)
	entryPredicates, err := b.entryPredicates()
	if err != nil {
		return nil, err
	}
	for _, predicate := range entryPredicates {
		predicate(selector)
	}
	return selector, nil
}

func (b *sumEntriesQuery) entryPredicates() ([]predicate.LedgerEntry, error) {
	entryPredicates := make([]predicate.LedgerEntry, 0, 4)
	entryPredicates = append(entryPredicates, ledgerentrydb.Namespace(b.query.Namespace))

	if b.query.Filters.TransactionID != nil {
		entryPredicates = append(entryPredicates, ledgerentrydb.TransactionID(*b.query.Filters.TransactionID))
	}

	entryPredicates = append(entryPredicates, entryProvenancePredicates(b.query.Filters.Provenance)...)

	if b.query.Filters.BookedAtPeriod != nil {
		transactionPredicates := make([]predicate.LedgerTransaction, 0, 2)
		if b.query.Filters.BookedAtPeriod.From != nil {
			transactionPredicates = append(transactionPredicates, ledgertransactiondb.BookedAtGTE(*b.query.Filters.BookedAtPeriod.From))
		}
		if b.query.Filters.BookedAtPeriod.To != nil {
			transactionPredicates = append(transactionPredicates, ledgertransactiondb.BookedAtLT(*b.query.Filters.BookedAtPeriod.To))
		}
		if len(transactionPredicates) > 0 {
			entryPredicates = append(entryPredicates, ledgerentrydb.HasTransactionWith(transactionPredicates...))
		}
	}

	if b.query.Filters.After != nil {
		after := b.query.Filters.After
		entryPredicates = append(entryPredicates, ledgerentrydb.HasTransactionWith(func(s *sql.Selector) {
			s.Where(sql.Or(
				sql.LT(s.C(ledgertransactiondb.FieldBookedAt), after.BookedAt),
				sql.And(
					sql.EQ(s.C(ledgertransactiondb.FieldBookedAt), after.BookedAt),
					sql.Or(
						sql.LT(s.C(ledgertransactiondb.FieldCreatedAt), after.CreatedAt),
						sql.And(
							sql.EQ(s.C(ledgertransactiondb.FieldCreatedAt), after.CreatedAt),
							sql.LTE(s.C(ledgertransactiondb.FieldID), after.ID.ID),
						),
					),
				),
			))
		}))
	}

	if b.query.Filters.AsOf != nil {
		entryPredicates = append(entryPredicates, ledgerentrydb.HasTransactionWith(
			ledgertransactiondb.BookedAtLTE(*b.query.Filters.AsOf),
		))
	}

	subAccountPredicates, err := b.subAccountPredicates()
	if err != nil {
		return nil, err
	}
	if len(subAccountPredicates) > 0 {
		entryPredicates = append(entryPredicates, ledgerentrydb.HasSubAccountWith(subAccountPredicates...))
	}

	return entryPredicates, nil
}

func (b *sumEntriesQuery) subAccountPredicates() ([]predicate.LedgerSubAccount, error) {
	subAccountPredicates := make([]predicate.LedgerSubAccount, 0, 1)
	if b.query.Filters.AccountID != nil {
		subAccountPredicates = append(subAccountPredicates, ledgersubaccountdb.AccountID(*b.query.Filters.AccountID))
	}
	normalizedRoute, err := b.query.Filters.Route.Normalize()
	if err != nil {
		return nil, ledger.ErrLedgerQueryInvalid.WithAttrs(models.Attributes{
			"reason": "route_invalid",
			"route":  b.query.Filters.Route,
			"error":  err,
		})
	}

	routePredicates := make([]predicate.LedgerSubAccountRoute, 0, 8)
	if normalizedRoute.Currency.Code != "" {
		if normalizedRoute.Currency.IsCustom() && !normalizedRoute.Currency.IsResolved() {
			prefix, err := normalizedRoute.Currency.MarshalTextPrefix()
			if err != nil {
				return nil, fmt.Errorf("serialize route currency filter prefix: %w", err)
			}
			routePredicates = append(routePredicates, ledgersubaccountroutedb.CurrencyHasPrefix(string(prefix)))
		} else {
			serialized, err := normalizedRoute.Currency.MarshalText()
			if err != nil {
				return nil, fmt.Errorf("serialize route currency filter: %w", err)
			}
			routePredicates = append(routePredicates, ledgersubaccountroutedb.Currency(string(serialized)))
		}
	}
	if normalizedRoute.CostBasisCurrency.IsPresent() {
		costBasisCurrency, _ := normalizedRoute.CostBasisCurrency.Get()
		if costBasisCurrency != nil {
			routePredicates = append(routePredicates, ledgersubaccountroutedb.CostBasisCurrency(*costBasisCurrency))
		} else {
			routePredicates = append(routePredicates, ledgersubaccountroutedb.CostBasisCurrencyIsNil())
		}
	}
	if normalizedRoute.CreditPriority != nil {
		routePredicates = append(routePredicates,
			ledgersubaccountroutedb.CreditPriority(*normalizedRoute.CreditPriority),
		)
	}
	if normalizedRoute.TaxCode.IsPresent() {
		tc, _ := normalizedRoute.TaxCode.Get()
		if tc != nil {
			routePredicates = append(routePredicates, ledgersubaccountroutedb.TaxCode(*tc))
		} else {
			routePredicates = append(routePredicates, ledgersubaccountroutedb.TaxCodeIsNil())
		}
	}
	if features, ok := normalizedRoute.Features.Get(); ok {
		routePredicates = append(routePredicates, func(s *sql.Selector) { s.Where(routequery.ExactFeaturesPredicate(s.C, features)) })
	}
	if normalizedRoute.MatchFeature != "" {
		routePredicates = append(routePredicates, func(s *sql.Selector) { s.Where(routequery.MatchFeaturePredicate(s.C, normalizedRoute.MatchFeature)) })
	}
	if normalizedRoute.CostBasis.IsPresent() {
		costBasis, _ := normalizedRoute.CostBasis.Get()
		if costBasis != nil {
			routePredicates = append(routePredicates, ledgersubaccountroutedb.CostBasis(*costBasis))
		} else {
			routePredicates = append(routePredicates, ledgersubaccountroutedb.CostBasisIsNil())
		}
	}
	if normalizedRoute.TaxBehavior.IsPresent() {
		tb, _ := normalizedRoute.TaxBehavior.Get()
		if tb != nil {
			routePredicates = append(routePredicates, ledgersubaccountroutedb.TaxBehavior(*tb))
		} else {
			routePredicates = append(routePredicates, ledgersubaccountroutedb.TaxBehaviorIsNil())
		}
	}
	if normalizedRoute.TransactionAuthorizationStatus != nil {
		routePredicates = append(routePredicates, ledgersubaccountroutedb.TransactionAuthorizationStatus(*normalizedRoute.TransactionAuthorizationStatus))
	}

	if len(routePredicates) > 0 {
		subAccountPredicates = append(subAccountPredicates, ledgersubaccountdb.HasRouteWith(routePredicates...))
	}

	return subAccountPredicates, nil
}

func entryProvenancePredicates(filter ledger.ProvenanceFilter) []predicate.LedgerEntry {
	entryPredicates := make([]predicate.LedgerEntry, 0, 3)

	if filter.SourceChargeID.IsPresent() {
		sourceChargeID, _ := filter.SourceChargeID.Get()
		if sourceChargeID != nil {
			entryPredicates = append(entryPredicates, ledgerentrydb.SourceChargeID(*sourceChargeID))
		} else {
			entryPredicates = append(entryPredicates, ledgerentrydb.SourceChargeIDIsNil())
		}
	}

	if filter.SpendChargeID.IsPresent() {
		spendChargeID, _ := filter.SpendChargeID.Get()
		if spendChargeID != nil {
			entryPredicates = append(entryPredicates, ledgerentrydb.SpendChargeID(*spendChargeID))
		} else {
			entryPredicates = append(entryPredicates, ledgerentrydb.SpendChargeIDIsNil())
		}
	}

	if filter.CollectionOriginID.IsPresent() {
		collectionOriginID, _ := filter.CollectionOriginID.Get()
		if collectionOriginID != nil {
			entryPredicates = append(entryPredicates, ledgerentrydb.CollectionOriginID(*collectionOriginID))
		} else {
			entryPredicates = append(entryPredicates, ledgerentrydb.CollectionOriginIDIsNil())
		}
	}

	return entryPredicates
}

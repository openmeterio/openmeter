package customerbalance

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
)

type correctionCreditTransactionLoader struct {
	service *service
}

func newCorrectionCreditTransactionLoader(s *service) creditTransactionLoader {
	return &correctionCreditTransactionLoader{service: s}
}

func (l *correctionCreditTransactionLoader) Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error) {
	items := make([]CreditTransaction, 0, input.Limit+1)
	after, before := input.After, input.Before
	resolved := make(map[string][]CreditTransaction)
	accountIDs := []string{input.AccountID}
	if input.ReceivableAccountID != "" {
		accountIDs = append(accountIDs, input.ReceivableAccountID)
	}

	for len(items) <= input.Limit {
		page, err := l.service.Ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
			Namespace: input.CustomerID.Namespace,
			Cursor:    after,
			Before:    before,
			Limit:     max(chargeListPageSize, input.Limit+1),
			EntryFilter: ledger.TransactionEntryFilter{
				AccountIDs: accountIDs,
				Currency:   input.Currency,
				Route:      featureFilterRoute(input.FeatureFilter),
			},
			ReturnOnlyMatchingEntries: true,
			AsOf:                      &input.AsOf,
			AnnotationFilters: map[string]string{
				ledger.AnnotationTransactionDirection: string(ledger.TransactionDirectionCorrection),
			},
			ExcludeAnnotationFilters: map[string]string{
				ledger.AnnotationCollectionType:            ledger.CollectionTypeBreakage,
				ledger.AnnotationCustomerBalanceVisibility: ledger.CustomerBalanceVisibilityInternal,
			},
		})
		if err != nil {
			return creditTransactionLoaderResult{}, err
		}
		if len(page.Items) == 0 {
			break
		}

		resume := page.Items[len(page.Items)-1].Cursor()
		if before != nil {
			resume = page.Items[0].Cursor()
			slices.Reverse(page.Items)
		}
		for _, candidate := range page.Items {
			groupID := candidate.GroupID()
			rows, ok := resolved[groupID.ID]
			if !ok {
				group, err := l.service.Ledger.GetTransactionGroup(ctx, groupID)
				if err != nil {
					return creditTransactionLoaderResult{}, fmt.Errorf("get correction group: %w", err)
				}
				rows = correctionCreditTransactions(group, input)
				resolved[groupID.ID] = rows
			}
			for _, row := range rows {
				if row.balanceCursor.Compare(candidate.Cursor()) == 0 && creditTransactionMatchesCursorWindow(row, input.After, input.Before) {
					items = append(items, row)
				}
			}
			if len(items) > input.Limit {
				break
			}
		}
		if page.NextCursor == nil {
			break
		}
		if before != nil {
			before = &resume
		} else {
			after = &resume
		}
	}

	hasMore := len(items) > input.Limit
	if hasMore {
		items = items[:input.Limit]
	}
	if input.Before != nil {
		slices.Reverse(items)
	}
	return creditTransactionLoaderResult{Items: items, HasMore: hasMore}, nil
}

// A correction can unwind collection, advance issuance, and later backing in
// separate transactions. Project their net balance effect together, without
// exposing temporary credit movements or requiring legacy allocation lineage.
func correctionCreditTransactions(group ledger.TransactionGroup, input creditTransactionLoaderInput) []CreditTransaction {
	type impactKey struct {
		bookedAt time.Time
		currency string
	}
	byImpact := make(map[impactKey]CreditTransaction)
	featureRoute := featureFilterRoute(input.FeatureFilter)
	for _, tx := range group.Transactions() {
		annotations := tx.Annotations()
		if annotations[ledger.AnnotationTransactionDirection] != string(ledger.TransactionDirectionCorrection) ||
			annotations[ledger.AnnotationCollectionType] == ledger.CollectionTypeBreakage ||
			annotations[ledger.AnnotationCustomerBalanceVisibility] == ledger.CustomerBalanceVisibilityInternal ||
			tx.BookedAt().After(input.AsOf) {
			continue
		}
		if _, voided := annotations[creditvoid.AnnotationCreditVoidRecordID]; voided {
			continue
		}

		// Net within each transaction first: canceling advance issuance changes
		// FBO and uncovered receivable by opposite amounts, with no balance effect.
		impacts := make(map[impactKey]CreditTransaction)
		for _, entry := range tx.Entries() {
			address := entry.PostingAddress()
			route := address.Route().Route()
			if address.AccountType() != ledger.AccountTypeCustomerFBO &&
				(address.AccountType() != ledger.AccountTypeCustomerReceivable || route.CostBasis != nil) {
				continue
			}
			if !route.Matches(featureRoute) || (input.Currency != nil && route.Currency.GetCode() != *input.Currency) {
				continue
			}
			key := impactKey{bookedAt: tx.BookedAt().UTC(), currency: route.Currency.IdentityKey()}
			row, ok := impacts[key]
			if !ok {
				cursor := tx.Cursor()
				row = CreditTransaction{
					ID:               tx.ID(),
					CreatedAt:        cursor.CreatedAt,
					BookedAt:         tx.BookedAt(),
					Type:             CreditTransactionTypeCorrection,
					Currency:         route.Currency.GetCode(),
					CustomCurrencyID: route.Currency.CustomCurrencyID,
					Amount:           alpacadecimal.Zero,
					Annotations:      annotations,
					balanceCursor:    &cursor,
				}
			}
			row.Amount = row.Amount.Add(entry.Amount())
			impacts[key] = row
		}
		for key, impact := range impacts {
			if impact.Amount.IsZero() {
				continue
			}
			row, ok := byImpact[key]
			if !ok {
				byImpact[key] = impact
				continue
			}
			amount := row.Amount.Add(impact.Amount)
			if impact.balanceCursor.Compare(*row.balanceCursor) > 0 {
				row = impact
			}
			row.Amount = amount
			byImpact[key] = row
		}
	}
	rows := make([]CreditTransaction, 0, len(byImpact))
	for _, row := range byImpact {
		if !row.Amount.IsZero() {
			rows = append(rows, row)
		}
	}
	slices.SortFunc(rows, func(a, b CreditTransaction) int { return -compareCreditTransactionsByCursor(a, b) })
	return rows
}

package customerbalance

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type fundedCreditTransactionLoader struct {
	service *service
}

func newFundedCreditTransactionLoader(s *service) creditTransactionLoader {
	return &fundedCreditTransactionLoader{service: s}
}

func (l *fundedCreditTransactionLoader) Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error) {
	result, err := l.service.CreditPurchaseSvc.ListFundedCreditActivities(ctx, creditpurchase.ListFundedCreditActivitiesInput{
		Customer:      input.CustomerID,
		Limit:         input.Limit,
		After:         toFundedCreditActivityCursor(input.After),
		Before:        toFundedCreditActivityCursor(input.Before),
		Currency:      input.Currency,
		AsOf:          &input.AsOf,
		FeatureFilter: input.FeatureFilter,
	})
	if err != nil {
		return creditTransactionLoaderResult{}, err
	}

	items := make([]CreditTransaction, 0, len(result.Items))
	for _, activity := range result.Items {
		annotations := models.Annotations{
			ledger.AnnotationChargeID: activity.ChargeID.ID,
		}

		items = append(items, CreditTransaction{
			ID:                       models.NamespacedID(activity.ChargeID),
			CreatedAt:                activity.ChargeCreatedAt,
			BookedAt:                 activity.FundedAt,
			Type:                     CreditTransactionTypeFunded,
			Currency:                 activity.Currency,
			Amount:                   activity.Amount,
			Name:                     activity.Name,
			Description:              activity.Description,
			Annotations:              annotations,
			fundedTransactionGroupID: activity.TransactionGroupID,
		})
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: result.NextCursor != nil,
	}, nil
}

// resolveBalances splits a funded transaction into separate balance movements
// when its ledger impacts have different booked times.
func (l *fundedCreditTransactionLoader) resolveBalances(
	ctx context.Context,
	input creditTransactionLoaderInput,
	item CreditTransaction,
) ([]CreditTransaction, error) {
	if item.fundedTransactionGroupID == "" {
		return []CreditTransaction{item}, nil
	}

	group, err := l.service.Ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: input.CustomerID.Namespace,
		ID:        item.fundedTransactionGroupID,
	})
	if err != nil {
		return nil, fmt.Errorf("get funded credit transaction group %s: %w", item.fundedTransactionGroupID, err)
	}

	impacts := fundedCreditTransactionBalanceImpacts(group, GetBalanceServiceInput{
		Currency:      item.Currency,
		FeatureFilter: input.FeatureFilter,
	})
	if len(impacts) == 0 {
		return nil, fmt.Errorf("funded credit transaction group %s has no customer balance impact", item.fundedTransactionGroupID)
	}

	total := alpacadecimal.Zero
	for _, impact := range impacts {
		total = total.Add(impact.Amount)
	}
	if !total.Equal(item.Amount) {
		return nil, fmt.Errorf(
			"funded credit transaction group %s customer balance impact %s does not match funded amount %s",
			item.fundedTransactionGroupID,
			total,
			item.Amount,
		)
	}

	items := make([]CreditTransaction, 0, len(impacts))
	for _, impact := range impacts {
		if impact.Cursor.BookedAt.After(input.AsOf) {
			continue
		}

		resolvedItem := item
		resolvedItem.BookedAt = impact.Cursor.BookedAt
		resolvedItem.Amount = impact.Amount
		resolvedItem.balanceCursor = &impact.Cursor
		resolvedItem.balanceImpact = &impact.Amount
		items = append(items, resolvedItem)
	}

	return items, nil
}

type fundedCreditTransactionBalanceImpact struct {
	Amount alpacadecimal.Decimal
	Cursor ledger.TransactionCursor
}

// fundedCreditTransactionBalanceImpacts follows the same two ledger components
// as settled balance and groups them by the time they affect that balance.
// Looking at the actual scoped entries keeps legacy credit purchase groups
// readable without relying on transaction template annotations.
func fundedCreditTransactionBalanceImpacts(group ledger.TransactionGroup, input GetBalanceServiceInput) []fundedCreditTransactionBalanceImpact {
	impactsByBookedAt := make(map[time.Time]fundedCreditTransactionBalanceImpact)

	for _, tx := range group.Transactions() {
		if tx.Annotations()[ledger.AnnotationCollectionType] == ledger.CollectionTypeBreakage {
			continue
		}

		impact := ledger.TransactionImpact(tx, ledger.ImpactFilter{
			AccountType: ledger.AccountTypeCustomerFBO,
			Route:       input.bookedRoute(),
		}).Add(ledger.TransactionImpact(tx, ledger.ImpactFilter{
			AccountType: ledger.AccountTypeCustomerReceivable,
			Route:       input.advanceRoute(),
		}))
		if impact.IsZero() {
			continue
		}

		cursor := tx.Cursor()
		bookedAt := cursor.BookedAt.UTC()
		groupedImpact, ok := impactsByBookedAt[bookedAt]
		if !ok {
			groupedImpact.Amount = alpacadecimal.Zero
		}
		groupedImpact.Amount = groupedImpact.Amount.Add(impact)
		if groupedImpact.Cursor.BookedAt.IsZero() || groupedImpact.Cursor.Compare(cursor) < 0 {
			groupedImpact.Cursor = cursor
		}
		impactsByBookedAt[bookedAt] = groupedImpact
	}

	impacts := make([]fundedCreditTransactionBalanceImpact, 0, len(impactsByBookedAt))
	for _, impact := range impactsByBookedAt {
		if !impact.Amount.IsZero() {
			impacts = append(impacts, impact)
		}
	}
	slices.SortFunc(impacts, func(a, b fundedCreditTransactionBalanceImpact) int {
		return b.Cursor.Compare(a.Cursor)
	})

	return impacts
}

func toFundedCreditActivityCursor(cursor *ledger.TransactionCursor) *creditpurchase.FundedCreditActivityCursor {
	if cursor == nil {
		return nil
	}

	return &creditpurchase.FundedCreditActivityCursor{
		FundedAt:        cursor.BookedAt,
		ChargeCreatedAt: cursor.CreatedAt,
		ChargeID:        chargesFundedCursorChargeID(cursor.ID),
	}
}

func chargesFundedCursorChargeID(id models.NamespacedID) meta.ChargeID {
	return meta.ChargeID{
		Namespace: id.Namespace,
		ID:        id.ID,
	}
}

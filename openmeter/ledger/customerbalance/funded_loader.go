package customerbalance

import (
	"context"
	"fmt"

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

func (l *fundedCreditTransactionLoader) resolveBalance(
	ctx context.Context,
	input creditTransactionLoaderInput,
	item *CreditTransaction,
) error {
	if item.fundedTransactionGroupID == "" {
		return nil
	}

	group, err := l.service.Ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: input.CustomerID.Namespace,
		ID:        item.fundedTransactionGroupID,
	})
	if err != nil {
		return fmt.Errorf("get funded credit transaction group %s: %w", item.fundedTransactionGroupID, err)
	}

	impact, cursor := fundedCreditTransactionBalanceImpact(group, GetBalanceServiceInput{
		Currency:      item.Currency,
		FeatureFilter: input.FeatureFilter,
	})
	if cursor == nil {
		return fmt.Errorf("funded credit transaction group %s has no customer balance impact", item.fundedTransactionGroupID)
	}
	if !impact.Equal(item.Amount) {
		return fmt.Errorf(
			"funded credit transaction group %s customer balance impact %s does not match funded amount %s",
			item.fundedTransactionGroupID,
			impact,
			item.Amount,
		)
	}

	item.balanceCursor = cursor
	item.balanceImpact = &impact

	return nil
}

// fundedCreditTransactionBalanceImpact follows the same two ledger components
// as settled balance. Looking at the actual scoped entries keeps legacy credit
// purchase groups readable without relying on transaction template annotations.
func fundedCreditTransactionBalanceImpact(group ledger.TransactionGroup, input GetBalanceServiceInput) (alpacadecimal.Decimal, *ledger.TransactionCursor) {
	total := alpacadecimal.Zero
	var latestCursor *ledger.TransactionCursor

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

		total = total.Add(impact)
		cursor := tx.Cursor()
		if latestCursor == nil || latestCursor.Compare(cursor) < 0 {
			latestCursor = &cursor
		}
	}

	return total, latestCursor
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

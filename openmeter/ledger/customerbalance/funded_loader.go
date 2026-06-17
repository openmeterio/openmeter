package customerbalance

import (
	"context"
	"fmt"

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
		balanceCursor, err := l.balanceCursorForFundedActivity(ctx, input.CustomerID.Namespace, activity)
		if err != nil {
			return creditTransactionLoaderResult{}, err
		}

		annotations := models.Annotations{
			ledger.AnnotationChargeID: activity.ChargeID.ID,
		}

		items = append(items, CreditTransaction{
			ID:            models.NamespacedID(activity.ChargeID),
			CreatedAt:     activity.ChargeCreatedAt,
			BookedAt:      activity.FundedAt,
			Type:          CreditTransactionTypeFunded,
			Currency:      activity.Currency,
			Amount:        activity.Amount,
			Name:          activity.Name,
			Description:   activity.Description,
			Annotations:   annotations,
			balanceCursor: balanceCursor,
		})
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: result.NextCursor != nil,
	}, nil
}

func (l *fundedCreditTransactionLoader) balanceCursorForFundedActivity(
	ctx context.Context,
	namespace string,
	activity creditpurchase.FundedCreditActivity,
) (*ledger.TransactionCursor, error) {
	if activity.TransactionGroupID == "" {
		return nil, nil
	}

	group, err := l.service.Ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        activity.TransactionGroupID,
	})
	if err != nil {
		return nil, fmt.Errorf("get funded credit transaction group %s: %w", activity.TransactionGroupID, err)
	}

	for _, tx := range group.Transactions() {
		impact, currency, err := creditTransactionFBOImpact(tx)
		if err != nil {
			continue
		}

		if currency == activity.Currency && impact.Equal(activity.Amount) {
			cursor := tx.Cursor()
			return &cursor, nil
		}
	}

	return nil, fmt.Errorf("funded credit transaction group %s has no matching customer FBO transaction", activity.TransactionGroupID)
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

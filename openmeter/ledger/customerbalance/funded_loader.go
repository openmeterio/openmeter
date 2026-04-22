package customerbalance

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
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
	result, err := l.service.ChargesService.ListFundedCreditActivities(ctx, charges.ListFundedCreditActivitiesInput{
		Customer: input.CustomerID,
		Limit:    input.Limit,
		After:    toFundedCreditActivityCursor(input.After),
		Before:   toFundedCreditActivityCursor(input.Before),
		Currency: input.Currency,
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
			ID:          models.NamespacedID(activity.ChargeID),
			CreatedAt:   activity.ChargeCreatedAt,
			BookedAt:    activity.FundedAt,
			Type:        CreditTransactionTypeFunded,
			Currency:    activity.Currency,
			Amount:      activity.Amount,
			Name:        activity.Name,
			Description: activity.Description,
			Annotations: annotations,
		})
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: result.NextCursor != nil,
	}, nil
}

func toFundedCreditActivityCursor(cursor *ledger.TransactionCursor) *charges.FundedCreditActivityCursor {
	if cursor == nil {
		return nil
	}

	return &charges.FundedCreditActivityCursor{
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

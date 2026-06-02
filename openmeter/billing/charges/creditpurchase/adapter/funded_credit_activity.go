package adapter

import (
	"context"
	"fmt"
	"slices"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	dbchargecreditpurchasecreditgrant "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchasecreditgrant"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

func (a *adapter) ListFundedCreditActivities(ctx context.Context, input creditpurchase.ListFundedCreditActivitiesInput) (creditpurchase.ListFundedCreditActivitiesResult, error) {
	return ListFundedCreditActivities(ctx, a.db, input)
}

func ListFundedCreditActivities(ctx context.Context, dbClient *db.Client, input creditpurchase.ListFundedCreditActivitiesInput) (creditpurchase.ListFundedCreditActivitiesResult, error) {
	query := dbClient.ChargeCreditPurchaseCreditGrant.Query().
		Where(
			dbchargecreditpurchasecreditgrant.Namespace(input.Customer.Namespace),
			dbchargecreditpurchasecreditgrant.DeletedAtIsNil(),
			dbchargecreditpurchasecreditgrant.HasCreditPurchaseWith(
				dbchargecreditpurchase.Namespace(input.Customer.Namespace),
				dbchargecreditpurchase.CustomerIDEQ(input.Customer.ID),
				dbchargecreditpurchase.DeletedAtIsNil(),
			),
		).
		WithCreditPurchase(func(q *db.ChargeCreditPurchaseQuery) {
			q.Where(
				dbchargecreditpurchase.Namespace(input.Customer.Namespace),
				dbchargecreditpurchase.DeletedAtIsNil(),
			)
		}).
		Limit(input.Limit + 1)

	if input.Before != nil {
		query = query.Order(
			dbchargecreditpurchasecreditgrant.ByGrantedAt(sql.OrderAsc()),
			dbchargecreditpurchasecreditgrant.ByCreditPurchaseField(dbchargecreditpurchase.FieldCreatedAt, sql.OrderAsc()),
			dbchargecreditpurchasecreditgrant.ByChargeID(sql.OrderAsc()),
		)
	} else {
		query = query.Order(
			dbchargecreditpurchasecreditgrant.ByGrantedAt(sql.OrderDesc()),
			dbchargecreditpurchasecreditgrant.ByCreditPurchaseField(dbchargecreditpurchase.FieldCreatedAt, sql.OrderDesc()),
			dbchargecreditpurchasecreditgrant.ByChargeID(sql.OrderDesc()),
		)
	}

	if input.Currency != nil {
		query = query.Where(
			dbchargecreditpurchasecreditgrant.HasCreditPurchaseWith(
				dbchargecreditpurchase.CurrencyEQ(*input.Currency),
			),
		)
	}

	if input.AsOf != nil {
		query = query.Where(dbchargecreditpurchasecreditgrant.GrantedAtLTE(*input.AsOf))
	}

	if input.After != nil {
		query = query.Where(fundedCreditActivityAfterPredicate(*input.After))
	}

	if input.Before != nil {
		query = query.Where(fundedCreditActivityBeforePredicate(*input.Before))
	}

	entities, err := query.All(ctx)
	if err != nil {
		return creditpurchase.ListFundedCreditActivitiesResult{}, fmt.Errorf("list funded credit activities: %w", err)
	}

	hasMore := len(entities) > input.Limit
	if hasMore {
		entities = entities[:input.Limit]
	}

	items := make([]creditpurchase.FundedCreditActivity, 0, len(entities))
	for _, entity := range entities {
		creditPurchase, err := entity.Edges.CreditPurchaseOrErr()
		if err != nil {
			return creditpurchase.ListFundedCreditActivitiesResult{}, fmt.Errorf("credit purchase not loaded for grant %s: %w", entity.ID, err)
		}

		items = append(items, creditpurchase.FundedCreditActivity{
			ChargeID: meta.ChargeID{
				Namespace: creditPurchase.Namespace,
				ID:        creditPurchase.ID,
			},
			ChargeCreatedAt:    creditPurchase.CreatedAt,
			FundedAt:           entity.GrantedAt,
			TransactionGroupID: entity.TransactionGroupID,
			Currency:           creditPurchase.Currency,
			Amount:             creditPurchase.CreditAmount,
			Name:               creditPurchase.Name,
			Description:        creditPurchase.Description,
		})
	}

	if input.Before != nil {
		slices.Reverse(items)
	}

	var nextCursor *creditpurchase.FundedCreditActivityCursor
	if hasMore && len(items) > 0 {
		next := items[len(items)-1]
		nextCursor = &creditpurchase.FundedCreditActivityCursor{
			FundedAt:        next.FundedAt,
			ChargeCreatedAt: next.ChargeCreatedAt,
			ChargeID:        next.ChargeID,
		}
	}

	hasPrevious := input.After != nil
	if input.Before != nil {
		hasPrevious = hasMore
	}

	return creditpurchase.ListFundedCreditActivitiesResult{
		Items:       items,
		NextCursor:  nextCursor,
		HasPrevious: hasPrevious,
	}, nil
}

func fundedCreditActivityAfterPredicate(cursor creditpurchase.FundedCreditActivityCursor) predicate.ChargeCreditPurchaseCreditGrant {
	return dbchargecreditpurchasecreditgrant.Or(
		dbchargecreditpurchasecreditgrant.GrantedAtLT(cursor.FundedAt),
		dbchargecreditpurchasecreditgrant.And(
			dbchargecreditpurchasecreditgrant.GrantedAtEQ(cursor.FundedAt),
			dbchargecreditpurchasecreditgrant.HasCreditPurchaseWith(
				dbchargecreditpurchase.CreatedAtLT(cursor.ChargeCreatedAt),
			),
		),
		dbchargecreditpurchasecreditgrant.And(
			dbchargecreditpurchasecreditgrant.GrantedAtEQ(cursor.FundedAt),
			dbchargecreditpurchasecreditgrant.HasCreditPurchaseWith(
				dbchargecreditpurchase.CreatedAtEQ(cursor.ChargeCreatedAt),
			),
			dbchargecreditpurchasecreditgrant.ChargeIDLT(cursor.ChargeID.ID),
		),
	)
}

func fundedCreditActivityBeforePredicate(cursor creditpurchase.FundedCreditActivityCursor) predicate.ChargeCreditPurchaseCreditGrant {
	return dbchargecreditpurchasecreditgrant.Or(
		dbchargecreditpurchasecreditgrant.GrantedAtGT(cursor.FundedAt),
		dbchargecreditpurchasecreditgrant.And(
			dbchargecreditpurchasecreditgrant.GrantedAtEQ(cursor.FundedAt),
			dbchargecreditpurchasecreditgrant.HasCreditPurchaseWith(
				dbchargecreditpurchase.CreatedAtGT(cursor.ChargeCreatedAt),
			),
		),
		dbchargecreditpurchasecreditgrant.And(
			dbchargecreditpurchasecreditgrant.GrantedAtEQ(cursor.FundedAt),
			dbchargecreditpurchasecreditgrant.HasCreditPurchaseWith(
				dbchargecreditpurchase.CreatedAtEQ(cursor.ChargeCreatedAt),
			),
			dbchargecreditpurchasecreditgrant.ChargeIDGT(cursor.ChargeID.ID),
		),
	)
}

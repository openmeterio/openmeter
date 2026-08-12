package adapter

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	dbchargecreditpurchasecostbasis "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchasecostbasis"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) loadCostBasisEdge(ctx context.Context, entity *entdb.ChargeCreditPurchase) error {
	if entity.Edges.CostBasis != nil {
		return fmt.Errorf("credit purchase cost basis edge is already loaded [charge_id=%s,edge_id=%s]", entity.ID, entity.Edges.CostBasis.ID)
	}

	if entity.CostBasisID == nil {
		return nil
	}

	costBasisEntity, err := a.db.ChargeCreditPurchaseCostBasis.Query().
		Where(
			dbchargecreditpurchasecostbasis.ID(*entity.CostBasisID),
			dbchargecreditpurchasecostbasis.Namespace(entity.Namespace),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("loading credit purchase cost basis edge [charge_id=%s,cost_basis_id=%s]: %w", entity.ID, *entity.CostBasisID, err)
	}

	entity.Edges.CostBasis = costBasisEntity

	return nil
}

func (a *adapter) SetResolvedCostBasis(ctx context.Context, input creditpurchase.SetResolvedCostBasisAdapterInput) (costbasis.State, error) {
	if err := input.Validate(); err != nil {
		return costbasis.State{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (costbasis.State, error) {
		_, err := tx.db.ChargeCreditPurchaseCostBasis.UpdateOneID(input.ChargeCostBasisID).
			Where(
				dbchargecreditpurchasecostbasis.Namespace(input.ChargeID.Namespace),
				costBasisOwnedByCharge(input.ChargeID),
			).
			SetOrClearResolvedCostBasisID(input.State.CostBasisID).
			SetResolvedCostBasis(input.State.CostBasis).
			SetResolvedAt(input.State.ResolvedAt.UTC()).
			Save(ctx)
		if entdb.IsNotFound(err) {
			return costbasis.State{}, models.NewGenericNotFoundError(
				fmt.Errorf(
					"credit purchase cost basis not found [charge_id=%s,cost_basis_id=%s]",
					input.ChargeID.ID,
					input.ChargeCostBasisID,
				),
			)
		}
		if err != nil {
			return costbasis.State{}, fmt.Errorf(
				"setting resolved credit purchase cost basis [charge_id=%s,cost_basis_id=%s]: %w",
				input.ChargeID.ID,
				input.ChargeCostBasisID,
				err,
			)
		}

		persistedState := input.State
		persistedState.ResolvedAt = persistedState.ResolvedAt.UTC()

		return persistedState, nil
	})
}

func costBasisOwnedByCharge(chargeID meta.ChargeID) predicate.ChargeCreditPurchaseCostBasis {
	return predicate.ChargeCreditPurchaseCostBasis(func(selector *sql.Selector) {
		chargeTable := sql.Table(dbchargecreditpurchase.Table)
		ownedCostBasisIDs := sql.Select(chargeTable.C(dbchargecreditpurchase.FieldCostBasisID)).
			From(chargeTable).
			Where(sql.And(
				sql.EQ(chargeTable.C(dbchargecreditpurchase.FieldID), chargeID.ID),
				sql.EQ(chargeTable.C(dbchargecreditpurchase.FieldNamespace), chargeID.Namespace),
			))

		selector.Where(sql.In(
			selector.C(dbchargecreditpurchasecostbasis.FieldID),
			ownedCostBasisIDs,
		))
	})
}

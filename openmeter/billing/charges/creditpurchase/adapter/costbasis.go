package adapter

import (
	"context"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargecreditpurchasecostbasis "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchasecostbasis"
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

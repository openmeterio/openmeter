package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfeecostbasis "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeecostbasis"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ flatfee.ChargeCostBasisAdapter = (*adapter)(nil)

func (a *adapter) loadCostBasisEdge(ctx context.Context, entity *entdb.ChargeFlatFee) error {
	if entity.Edges.CostBasis != nil {
		return fmt.Errorf("flat fee cost basis edge is already loaded [charge_id=%s,edge_id=%s]", entity.ID, entity.Edges.CostBasis.ID)
	}

	if entity.CostBasisID == nil {
		return nil
	}

	costBasisEntity, err := a.db.ChargeFlatFeeCostBasis.Query().
		Where(
			dbchargeflatfeecostbasis.ID(*entity.CostBasisID),
			dbchargeflatfeecostbasis.Namespace(entity.Namespace),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("load flat fee cost basis edge [charge_id=%s,cost_basis_id=%s]: %w", entity.ID, *entity.CostBasisID, err)
	}

	entity.Edges.CostBasis = costBasisEntity

	return nil
}

func (a *adapter) SetResolvedCostBasis(ctx context.Context, input costbasis.SetResolvedCostBasisInput) (costbasis.CostBasis, error) {
	if err := input.Validate(); err != nil {
		return costbasis.CostBasis{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (costbasis.CostBasis, error) {
		update := tx.db.ChargeFlatFeeCostBasis.UpdateOneID(input.ID).
			Where(
				dbchargeflatfeecostbasis.Namespace(input.Namespace),
				dbchargeflatfeecostbasis.ModeEQ(costbasis.ModeDynamic),
				dbchargeflatfeecostbasis.ResolvedCostBasisIDIsNil(),
				dbchargeflatfeecostbasis.ResolvedCostBasisIsNil(),
				dbchargeflatfeecostbasis.ResolvedAtIsNil(),
			)

		update, err := costbasis.Set(update, input.State)
		if err != nil {
			return costbasis.CostBasis{}, err
		}

		entity, err := update.Save(ctx)
		if entdb.IsNotFound(err) {
			existingEntity, getErr := tx.db.ChargeFlatFeeCostBasis.Query().
				Where(
					dbchargeflatfeecostbasis.ID(input.ID),
					dbchargeflatfeecostbasis.Namespace(input.Namespace),
				).
				Only(ctx)
			if entdb.IsNotFound(getErr) {
				return costbasis.CostBasis{}, models.NewGenericNotFoundError(
					fmt.Errorf("flat fee cost basis not found: %s", input.ID),
				)
			}
			if getErr != nil {
				return costbasis.CostBasis{}, fmt.Errorf("get flat fee cost basis after conditional resolution: %w", getErr)
			}

			existing, getErr := costbasis.Get(existingEntity)
			if getErr != nil {
				return costbasis.CostBasis{}, fmt.Errorf("map flat fee cost basis after conditional resolution: %w", getErr)
			}

			if existing.Intent.Kind() == costbasis.ModeDynamic && existing.State != nil {
				return existing, nil
			}

			return costbasis.CostBasis{}, models.NewGenericValidationError(
				fmt.Errorf("flat fee cost basis is not an unresolved dynamic cost basis: %s", input.ID),
			)
		}
		if err != nil {
			return costbasis.CostBasis{}, fmt.Errorf("set resolved flat fee cost basis: %w", err)
		}

		return costbasis.Get(entity)
	})
}

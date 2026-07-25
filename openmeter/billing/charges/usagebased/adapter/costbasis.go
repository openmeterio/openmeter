package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeusagebasedcostbasis "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedcostbasis"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ usagebased.ChargeCostBasisAdapter = (*adapter)(nil)

func (a *adapter) loadCostBasisEdge(ctx context.Context, entity *entdb.ChargeUsageBased) error {
	if entity.Edges.CostBasis != nil {
		return fmt.Errorf("usage based cost basis edge is already loaded [charge_id=%s,edge_id=%s]", entity.ID, entity.Edges.CostBasis.ID)
	}

	if entity.CostBasisID == nil {
		return nil
	}

	costBasisEntity, err := a.db.ChargeUsageBasedCostBasis.Query().
		Where(
			dbchargeusagebasedcostbasis.ID(*entity.CostBasisID),
			dbchargeusagebasedcostbasis.Namespace(entity.Namespace),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("load usage based cost basis edge [charge_id=%s,cost_basis_id=%s]: %w", entity.ID, *entity.CostBasisID, err)
	}

	entity.Edges.CostBasis = costBasisEntity

	return nil
}

func (a *adapter) SetResolvedCostBasis(ctx context.Context, input costbasis.SetResolvedCostBasisInput) (costbasis.CostBasis, error) {
	if err := input.Validate(); err != nil {
		return costbasis.CostBasis{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (costbasis.CostBasis, error) {
		update := tx.db.ChargeUsageBasedCostBasis.UpdateOneID(input.ID).
			Where(
				dbchargeusagebasedcostbasis.Namespace(input.Namespace),
				dbchargeusagebasedcostbasis.ModeEQ(costbasis.ModeDynamic),
				dbchargeusagebasedcostbasis.ResolvedCostBasisIDIsNil(),
				dbchargeusagebasedcostbasis.ResolvedCostBasisIsNil(),
				dbchargeusagebasedcostbasis.ResolvedAtIsNil(),
			)

		update, err := costbasis.Set(update, input.State)
		if err != nil {
			return costbasis.CostBasis{}, err
		}

		entity, err := update.Save(ctx)
		if entdb.IsNotFound(err) {
			existingEntity, getErr := tx.db.ChargeUsageBasedCostBasis.Query().
				Where(
					dbchargeusagebasedcostbasis.ID(input.ID),
					dbchargeusagebasedcostbasis.Namespace(input.Namespace),
				).
				Only(ctx)
			if entdb.IsNotFound(getErr) {
				return costbasis.CostBasis{}, models.NewGenericNotFoundError(
					fmt.Errorf("usage based cost basis not found: %s", input.ID),
				)
			}
			if getErr != nil {
				return costbasis.CostBasis{}, fmt.Errorf("get usage based cost basis after conditional resolution: %w", getErr)
			}

			existing, getErr := costbasis.Get(existingEntity)
			if getErr != nil {
				return costbasis.CostBasis{}, fmt.Errorf("map usage based cost basis after conditional resolution: %w", getErr)
			}

			if existing.Intent.Kind() == costbasis.ModeDynamic && existing.State != nil {
				return existing, nil
			}

			return costbasis.CostBasis{}, models.NewGenericValidationError(
				fmt.Errorf("usage based cost basis is not an unresolved dynamic cost basis: %s", input.ID),
			)
		}
		if err != nil {
			return costbasis.CostBasis{}, fmt.Errorf("set resolved usage based cost basis: %w", err)
		}

		return costbasis.Get(entity)
	})
}

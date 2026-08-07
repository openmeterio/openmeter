package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	dbchargecreditpurchasecostbasis "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchasecostbasis"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// upgradeCostBasisSchemaForWrite materializes the legacy settlement cost basis
// while holding a row lock. Callers must run it inside the transaction that
// performs the business write so schema level 2 becomes visible atomically.
func (a *adapter) upgradeCostBasisSchemaForWrite(ctx context.Context, chargeID meta.ChargeID) (*entdb.ChargeCreditPurchase, error) {
	entity, err := a.db.ChargeCreditPurchase.Query().
		Where(
			dbchargecreditpurchase.Namespace(chargeID.Namespace),
			dbchargecreditpurchase.ID(chargeID.ID),
		).
		WithCostBasis().
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("locking credit purchase charge for cost-basis schema upgrade [id=%s]: %w", chargeID.ID, err)
	}

	switch creditpurchase.SchemaLevel(entity.SchemaLevel) {
	case creditpurchase.SchemaLevelCostBasis:
		return entity, nil
	case creditpurchase.SchemaLevelLegacy:
		if entity.FiatCostBasis != nil || entity.CostBasisID != nil || entity.Edges.CostBasis != nil ||
			entity.SettlementType != nil || entity.InitialPaymentSettlementStatus != nil {
			return nil, errors.New("legacy credit purchase contains dedicated schema-level state")
		}
	default:
		return nil, fmt.Errorf("unsupported credit purchase schema level: %d", entity.SchemaLevel)
	}

	update := a.db.ChargeCreditPurchase.UpdateOneID(entity.ID).
		Where(
			dbchargecreditpurchase.Namespace(entity.Namespace),
			dbchargecreditpurchase.SchemaLevelEQ(int(creditpurchase.SchemaLevelLegacy)),
		).
		SetSettlementType(entity.Settlement.Type)

	if entity.Settlement.Type == creditpurchase.SettlementTypeExternal {
		if entity.Settlement.InitialStatus == nil {
			return nil, errors.New("legacy external settlement requires an initial payment settlement status")
		}
		update.SetInitialPaymentSettlementStatus(*entity.Settlement.InitialStatus)
	}

	switch entity.Settlement.Type {
	case creditpurchase.SettlementTypePromotional:
		// Promotional purchases have no cost basis to materialize.
	case creditpurchase.SettlementTypeInvoice, creditpurchase.SettlementTypeExternal:
		rate, err := entity.Settlement.GetCostBasis()
		if err != nil {
			return nil, fmt.Errorf("getting legacy settlement cost basis: %w", err)
		}

		if entity.CustomCurrencyID == nil {
			update.SetFiatCostBasis(rate)
			break
		}

		fiatCode, err := entity.Settlement.GetCurrency()
		if err != nil {
			return nil, fmt.Errorf("getting legacy settlement currency: %w", err)
		}
		if fiatCode == nil {
			return nil, errors.New("legacy custom-currency cost basis requires a settlement currency")
		}

		fiatCurrency, err := currencyx.NewFiatCurrency(*fiatCode)
		if err != nil {
			return nil, fmt.Errorf("mapping legacy settlement currency: %w", err)
		}

		costBasisEntity, err := costbasis.Create(a.db.ChargeCreditPurchaseCostBasis.Create(), costbasis.CreateInput{
			NamespacedID: models.NamespacedID{
				Namespace: entity.Namespace,
				ID:        ulid.Make().String(),
			},
			CurrencyID: *entity.CustomCurrencyID,
			Intent: costbasis.NewIntent(costbasis.ManualIntent{
				FiatCurrency: fiatCurrency,
				Rate:         rate,
			}),
			State: &costbasis.State{
				CostBasis:  rate,
				ResolvedAt: entity.CreatedAt.UTC(),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("building legacy custom-currency cost basis: %w", err)
		}

		createdCostBasis, err := costBasisEntity.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("creating legacy custom-currency cost basis: %w", err)
		}

		update.SetCostBasisID(createdCostBasis.ID)
		entity.Edges.CostBasis = createdCostBasis
	default:
		return nil, fmt.Errorf("unsupported credit purchase settlement type: %s", entity.Settlement.Type)
	}

	upgraded, err := update.
		SetSchemaLevel(int(creditpurchase.SchemaLevelCostBasis)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("upgrading credit purchase cost-basis schema [id=%s]: %w", entity.ID, err)
	}

	upgraded.Edges.CostBasis = entity.Edges.CostBasis

	return upgraded, nil
}

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

package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type mapLegacyCostBasisInput struct {
	Settlement             creditpurchase.PersistedSettlement
	CustomCurrency         bool
	FiatCreditCurrencyCode *currencyx.Code
	CreatedAt              time.Time
}

var _ models.Validator = (*mapLegacyCostBasisInput)(nil)

func (i mapLegacyCostBasisInput) Validate() error {
	var errs []error

	if err := i.Settlement.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement: %w", err))
	}

	if i.CustomCurrency {
		if i.FiatCreditCurrencyCode != nil {
			errs = append(errs, errors.New("custom-currency charge cannot have a fiat credit currency code"))
		}
	} else if i.FiatCreditCurrencyCode == nil {
		errs = append(errs, errors.New("fiat credit currency code is required"))
	} else if !i.FiatCreditCurrencyCode.IsFiat() {
		errs = append(errs, fmt.Errorf("credit currency %q is not fiat", *i.FiatCreditCurrencyCode))
	}

	if i.CustomCurrency && i.Settlement.Type != creditpurchase.SettlementTypePromotional && i.CreatedAt.IsZero() {
		errs = append(errs, errors.New("created at is required for custom-currency cost basis"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func mapLegacyCostBasis(in mapLegacyCostBasisInput) (mappedCostBasis, error) {
	if err := in.Validate(); err != nil {
		return mappedCostBasis{}, err
	}

	if in.Settlement.Type == creditpurchase.SettlementTypePromotional {
		return mappedCostBasis{}, nil
	}

	rate, err := in.Settlement.GetCostBasis()
	if err != nil {
		return mappedCostBasis{}, fmt.Errorf("getting legacy settlement cost basis: %w", err)
	}

	fiatCode, err := in.Settlement.GetCurrency()
	if err != nil {
		return mappedCostBasis{}, fmt.Errorf("getting legacy settlement currency: %w", err)
	}
	if fiatCode == nil {
		return mappedCostBasis{}, errors.New("legacy cost basis requires a settlement currency")
	}

	if !in.CustomCurrency {
		if currencyx.Code(*fiatCode) != *in.FiatCreditCurrencyCode {
			return mappedCostBasis{}, fmt.Errorf("settlement currency %q must match credit currency %q", *fiatCode, *in.FiatCreditCurrencyCode)
		}

		return mappedCostBasis{
			CostBasis: lo.ToPtr(creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: rate})),
		}, nil
	}

	fiatCurrency, err := currencyx.NewFiatCurrency(*fiatCode)
	if err != nil {
		return mappedCostBasis{}, fmt.Errorf("mapping legacy settlement currency: %w", err)
	}

	intent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         rate,
	})
	state := costbasis.State{
		CostBasis:  rate,
		ResolvedAt: in.CreatedAt.UTC(),
	}

	return mappedCostBasis{
		CostBasis:         lo.ToPtr(creditpurchase.NewCostBasis(intent)),
		ResolvedCostBasis: &state,
	}, nil
}

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

	if entity.CustomCurrencyID != nil && entity.FiatCurrencyCode != nil {
		return nil, errors.New("legacy credit purchase contains both fiat and custom currency state")
	}

	mappedLegacyCostBasis, err := mapLegacyCostBasis(mapLegacyCostBasisInput{
		Settlement:             entity.Settlement,
		CustomCurrency:         entity.CustomCurrencyID != nil,
		FiatCreditCurrencyCode: entity.FiatCurrencyCode,
		CreatedAt:              entity.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("mapping legacy cost basis: %w", err)
	}

	if mappedLegacyCostBasis.CostBasis != nil {
		switch mappedLegacyCostBasis.CostBasis.Type() {
		case creditpurchase.CostBasisTypeFiat:
			fiatCostBasis, err := mappedLegacyCostBasis.CostBasis.AsFiat()
			if err != nil {
				return nil, fmt.Errorf("getting mapped legacy fiat cost basis: %w", err)
			}

			update.SetFiatCostBasis(fiatCostBasis.Rate)
		case creditpurchase.CostBasisTypeCustomCurrency:
			if entity.CustomCurrencyID == nil {
				return nil, errors.New("legacy custom-currency cost basis requires a custom currency ID")
			}
			if mappedLegacyCostBasis.ResolvedCostBasis == nil {
				return nil, errors.New("mapped legacy custom-currency cost basis is unresolved")
			}

			intent, err := mappedLegacyCostBasis.CostBasis.AsCustomCurrency()
			if err != nil {
				return nil, fmt.Errorf("getting mapped legacy custom-currency cost basis: %w", err)
			}

			costBasisEntity, err := costbasis.Create(a.db.ChargeCreditPurchaseCostBasis.Create(), costbasis.CreateInput{
				NamespacedID: models.NamespacedID{
					Namespace: entity.Namespace,
					ID:        ulid.Make().String(),
				},
				CurrencyID: *entity.CustomCurrencyID,
				Intent:     intent,
				State:      mappedLegacyCostBasis.ResolvedCostBasis,
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
			return nil, fmt.Errorf("unsupported mapped legacy cost basis type: %s", mappedLegacyCostBasis.CostBasis.Type())
		}
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

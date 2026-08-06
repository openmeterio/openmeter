package adapter

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func fromDBBaseWithCurrency(dbEntity *entdb.ChargeCreditPurchase, currency currencies.Currency) (creditpurchase.ChargeBase, error) {
	mappedMeta, err := chargemeta.FromDBWithCurrency(dbEntity, currency)
	if err != nil {
		return creditpurchase.ChargeBase{}, fmt.Errorf("failed to map charge base: %w", err)
	}

	return fromDBBase(dbEntity, mappedMeta)
}

func fromDBBase(dbEntity *entdb.ChargeCreditPurchase, mappedMeta meta.Charge) (creditpurchase.ChargeBase, error) {
	mappedSettlement, err := fromDBSettlement(dbEntity)
	if err != nil {
		return creditpurchase.ChargeBase{}, fmt.Errorf("mapping credit purchase settlement [id=%s]: %w", dbEntity.ID, err)
	}

	mappedCostBasis, err := fromDBCostBasis(dbEntity, mappedMeta.Intent.Currency)
	if err != nil {
		return creditpurchase.ChargeBase{}, fmt.Errorf("mapping credit purchase cost basis [id=%s]: %w", dbEntity.ID, err)
	}

	charge := creditpurchase.ChargeBase{
		ManagedResource: mappedMeta.ManagedResource,
		Status:          dbEntity.StatusDetailed,
		Intent: creditpurchase.Intent{
			Intent: mappedMeta.Intent,
			IntentMutableFields: creditpurchase.IntentMutableFields{
				IntentMutableFields: mappedMeta.IntentMutableFields,
				CreditAmount:        dbEntity.CreditAmount,
				EffectiveAt:         convert.SafeToUTC(dbEntity.EffectiveAt),
				ExpiresAt:           convert.SafeToUTC(dbEntity.ExpiresAt),
				Priority:            dbEntity.Priority,
				FeatureFilters:      creditpurchase.FeatureFilters(dbEntity.FeatureFilters).Normalize(),
				Settlement:          mappedSettlement,
			},
			CostBasis: mappedCostBasis.CostBasis,
			Key:       dbEntity.Key,
		},
		State: creditpurchase.State{
			SchemaLevel:       creditpurchase.SchemaLevel(dbEntity.SchemaLevel),
			VoidedAt:          convert.SafeToUTC(dbEntity.VoidedAt),
			CostBasisID:       mappedCostBasis.CostBasisID,
			ResolvedCostBasis: mappedCostBasis.ResolvedCostBasis,
		},
	}

	if err := charge.Validate(); err != nil {
		return creditpurchase.ChargeBase{}, fmt.Errorf("validating mapped credit purchase charge [id=%s]: %w", dbEntity.ID, err)
	}

	return charge, nil
}

func fromDBSettlement(dbEntity *entdb.ChargeCreditPurchase) (creditpurchase.Settlement, error) {
	schemaLevel := creditpurchase.SchemaLevel(dbEntity.SchemaLevel)
	if err := schemaLevel.Validate(); err != nil {
		return creditpurchase.Settlement{}, err
	}

	switch schemaLevel {
	case creditpurchase.SchemaLevelLegacy:
		if dbEntity.SettlementType != nil || dbEntity.InitialPaymentSettlementStatus != nil {
			return creditpurchase.Settlement{}, errors.New("legacy schema row contains dedicated settlement state")
		}

		return dbEntity.Settlement.AsSettlement()
	case creditpurchase.SchemaLevelCostBasis:
		if dbEntity.SettlementType == nil {
			return creditpurchase.Settlement{}, errors.New("settlement type is required")
		}

		switch *dbEntity.SettlementType {
		case creditpurchase.SettlementTypeInvoice:
			if dbEntity.InitialPaymentSettlementStatus != nil {
				return creditpurchase.Settlement{}, errors.New("invoice settlement cannot have an initial payment settlement status")
			}

			return creditpurchase.NewInvoiceSettlement(), nil
		case creditpurchase.SettlementTypeExternal:
			if dbEntity.InitialPaymentSettlementStatus == nil {
				return creditpurchase.Settlement{}, errors.New("external settlement requires an initial payment settlement status")
			}

			return creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: *dbEntity.InitialPaymentSettlementStatus,
			}), nil
		case creditpurchase.SettlementTypePromotional:
			if dbEntity.InitialPaymentSettlementStatus != nil {
				return creditpurchase.Settlement{}, errors.New("promotional settlement cannot have an initial payment settlement status")
			}

			return creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}), nil
		default:
			return creditpurchase.Settlement{}, fmt.Errorf("unsupported credit purchase settlement type: %s", *dbEntity.SettlementType)
		}
	default:
		return creditpurchase.Settlement{}, fmt.Errorf("unsupported credit purchase schema level: %d", schemaLevel)
	}
}

type mappedCostBasis struct {
	CostBasis         *creditpurchase.CostBasis
	CostBasisID       *string
	ResolvedCostBasis *costbasis.State
}

func fromDBCostBasis(dbEntity *entdb.ChargeCreditPurchase, currency currencies.Currency) (mappedCostBasis, error) {
	schemaLevel := creditpurchase.SchemaLevel(dbEntity.SchemaLevel)
	if err := schemaLevel.Validate(); err != nil {
		return mappedCostBasis{}, err
	}

	switch schemaLevel {
	case creditpurchase.SchemaLevelLegacy:
		if dbEntity.FiatCostBasis != nil || dbEntity.CostBasisID != nil || dbEntity.Edges.CostBasis != nil {
			return mappedCostBasis{}, errors.New("legacy schema row contains dedicated cost-basis state")
		}

		return fromDBLegacyCostBasis(dbEntity, currency)
	case creditpurchase.SchemaLevelCostBasis:
		return fromDBDedicatedCostBasis(dbEntity, currency)
	default:
		return mappedCostBasis{}, fmt.Errorf("unsupported credit purchase schema level: %d", schemaLevel)
	}
}

func fromDBLegacyCostBasis(dbEntity *entdb.ChargeCreditPurchase, currency currencies.Currency) (mappedCostBasis, error) {
	if dbEntity.Settlement.Type == creditpurchase.SettlementTypePromotional {
		return mappedCostBasis{}, nil
	}

	rate, err := dbEntity.Settlement.GetCostBasis()
	if err != nil {
		return mappedCostBasis{}, fmt.Errorf("getting legacy settlement cost basis: %w", err)
	}

	if !currency.IsCustom() {
		fiatCode, err := dbEntity.Settlement.GetCurrency()
		if err != nil {
			return mappedCostBasis{}, fmt.Errorf("getting legacy settlement currency: %w", err)
		}
		if fiatCode == nil {
			return mappedCostBasis{}, errors.New("legacy fiat cost basis requires a settlement currency")
		}
		if currencyx.Code(*fiatCode) != currency.GetCode() {
			return mappedCostBasis{}, fmt.Errorf("settlement currency %q must match credit currency %q", *fiatCode, currency.GetCode())
		}

		return mappedCostBasis{CostBasis: lo.ToPtr(creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: rate}))}, nil
	}

	fiatCode, err := dbEntity.Settlement.GetCurrency()
	if err != nil {
		return mappedCostBasis{}, fmt.Errorf("getting legacy settlement currency: %w", err)
	}
	if fiatCode == nil {
		return mappedCostBasis{}, errors.New("legacy custom-currency cost basis requires a settlement currency")
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
		ResolvedAt: dbEntity.CreatedAt.UTC(),
	}

	return mappedCostBasis{
		CostBasis:         lo.ToPtr(creditpurchase.NewCostBasis(intent)),
		ResolvedCostBasis: &state,
	}, nil
}

func fromDBDedicatedCostBasis(dbEntity *entdb.ChargeCreditPurchase, currency currencies.Currency) (mappedCostBasis, error) {
	if dbEntity.SettlementType == nil {
		return mappedCostBasis{}, errors.New("settlement type is required")
	}

	if *dbEntity.SettlementType == creditpurchase.SettlementTypePromotional {
		if dbEntity.FiatCostBasis != nil || dbEntity.CostBasisID != nil || dbEntity.Edges.CostBasis != nil {
			return mappedCostBasis{}, errors.New("promotional credit purchase contains dedicated cost-basis state")
		}

		return mappedCostBasis{}, nil
	}

	if !currency.IsCustom() {
		if dbEntity.FiatCostBasis == nil {
			return mappedCostBasis{}, errors.New("fiat cost basis is required")
		}
		if dbEntity.CostBasisID != nil || dbEntity.Edges.CostBasis != nil {
			return mappedCostBasis{}, errors.New("fiat credit purchase contains custom-currency cost-basis state")
		}

		return mappedCostBasis{CostBasis: lo.ToPtr(creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: *dbEntity.FiatCostBasis}))}, nil
	}

	if dbEntity.FiatCostBasis != nil {
		return mappedCostBasis{}, errors.New("custom-currency credit purchase contains fiat cost-basis state")
	}
	if dbEntity.CostBasisID == nil {
		return mappedCostBasis{}, errors.New("custom-currency cost basis ID is required")
	}
	if dbEntity.Edges.CostBasis == nil {
		return mappedCostBasis{}, fmt.Errorf("custom-currency cost basis not loaded [cost_basis_id=%s]", *dbEntity.CostBasisID)
	}
	if dbEntity.Edges.CostBasis.ID != *dbEntity.CostBasisID {
		return mappedCostBasis{}, fmt.Errorf("custom-currency cost basis ID mismatch [cost_basis_id=%s,edge_id=%s]", *dbEntity.CostBasisID, dbEntity.Edges.CostBasis.ID)
	}

	mappedCustomCostBasis, err := costbasis.Get(dbEntity.Edges.CostBasis)
	if err != nil {
		return mappedCostBasis{}, fmt.Errorf("mapping custom-currency cost basis: %w", err)
	}
	if mappedCustomCostBasis.CurrencyID != currency.ID {
		return mappedCostBasis{}, fmt.Errorf("custom-currency cost basis currency mismatch [currency_id=%s,cost_basis_currency_id=%s]", currency.ID, mappedCustomCostBasis.CurrencyID)
	}
	if mappedCustomCostBasis.State == nil {
		return mappedCostBasis{}, errors.New("custom-currency cost basis is unresolved")
	}

	return mappedCostBasis{
		CostBasis:         lo.ToPtr(creditpurchase.NewCostBasis(mappedCustomCostBasis.Intent)),
		CostBasisID:       lo.ToPtr(*dbEntity.CostBasisID),
		ResolvedCostBasis: mappedCustomCostBasis.State,
	}, nil
}

func FromDB(dbEntity *entdb.ChargeCreditPurchase, expands meta.Expands) (creditpurchase.Charge, error) {
	mappedMeta, err := chargemeta.FromDB(dbEntity, dbEntity.Edges)
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("failed to map charge meta: %w", err)
	}

	return fromDBWithMeta(dbEntity, mappedMeta, expands)
}

func FromDBWithCurrency(dbEntity *entdb.ChargeCreditPurchase, currency currencies.Currency, expands meta.Expands) (creditpurchase.Charge, error) {
	mappedMeta, err := chargemeta.FromDBWithCurrency(dbEntity, currency)
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("failed to map charge meta: %w", err)
	}

	return fromDBWithMeta(dbEntity, mappedMeta, expands)
}

func fromDBWithMeta(dbEntity *entdb.ChargeCreditPurchase, mappedMeta meta.Charge, expands meta.Expands) (creditpurchase.Charge, error) {
	chargeBase, err := fromDBBase(dbEntity, mappedMeta)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	var creditGrantRealization *ledgertransaction.TimedGroupReference
	var externalPaymentSettlement *payment.External
	var invoiceSettlement *payment.Invoiced
	if expands.Has(meta.ExpandRealizations) {
		dbCreditGrant, err := dbEntity.Edges.CreditGrantOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return creditpurchase.Charge{}, fmt.Errorf("credit grant not loaded for credit purchase charge [id=%s]: %w", dbEntity.ID, err)
		}

		if dbCreditGrant != nil {
			creditGrantRealization = &ledgertransaction.TimedGroupReference{
				GroupReference: ledgertransaction.GroupReference{
					TransactionGroupID: dbCreditGrant.TransactionGroupID,
				},
				Time: dbCreditGrant.GrantedAt.In(time.UTC),
			}
		}

		dbExternalPaymentSettlement, err := dbEntity.Edges.ExternalPaymentOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return creditpurchase.Charge{}, fmt.Errorf("external payment settlement not loaded for credit purchase charge [id=%s]: %w", dbEntity.ID, err)
		}

		if dbExternalPaymentSettlement != nil {
			externalPaymentSettlement = lo.ToPtr(payment.MapExternalFromDB(dbExternalPaymentSettlement))
		}

		dbInvoicedPaymentSettlement, err := dbEntity.Edges.InvoicedPaymentOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return creditpurchase.Charge{}, fmt.Errorf("invoiced payment settlement not loaded for credit purchase charge [id=%s]: %w", dbEntity.ID, err)
		}

		if dbInvoicedPaymentSettlement != nil {
			invoiceSettlement = lo.ToPtr(payment.MapInvoicedFromDB(dbInvoicedPaymentSettlement))
		}
	}

	return creditpurchase.Charge{
		ChargeBase: chargeBase,
		Realizations: creditpurchase.Realizations{
			CreditGrantRealization:    creditGrantRealization,
			ExternalPaymentSettlement: externalPaymentSettlement,
			InvoiceSettlement:         invoiceSettlement,
		},
	}, nil
}

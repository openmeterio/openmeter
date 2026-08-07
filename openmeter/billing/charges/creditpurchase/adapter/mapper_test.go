package adapter

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestFromDBCostBasis(t *testing.T) {
	createdAt := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	rate := alpacadecimal.NewFromFloat(0.5)
	fiatCode := currencyx.FiatCode("USD")
	invoiceSettlementType := creditpurchase.SettlementTypeInvoice
	settlement := creditpurchase.PersistedSettlement{
		Type:      creditpurchase.SettlementTypeInvoice,
		Currency:  &fiatCode,
		CostBasis: &rate,
	}

	t.Run("legacy fiat is projected from settlement", func(t *testing.T) {
		mapped, err := fromDBCostBasis(&db.ChargeCreditPurchase{
			SchemaLevel: int(creditpurchase.SchemaLevelLegacy),
			Settlement:  settlement,
			CreatedAt:   createdAt,
		}, currenciestestutils.NewFiatCurrency(t, "USD"))
		require.NoError(t, err)
		require.Nil(t, mapped.CostBasisID)
		require.Nil(t, mapped.ResolvedCostBasis)
		require.NotNil(t, mapped.CostBasis)

		fiat, err := mapped.CostBasis.AsFiat()
		require.NoError(t, err)
		require.Equal(t, 0.5, fiat.Rate.InexactFloat64())
	})

	t.Run("legacy fiat rejects mismatched settlement currency", func(t *testing.T) {
		eur := currencyx.FiatCode("EUR")
		mismatchedSettlement := settlement
		mismatchedSettlement.Currency = &eur

		_, err := fromDBCostBasis(&db.ChargeCreditPurchase{
			SchemaLevel: int(creditpurchase.SchemaLevelLegacy),
			Settlement:  mismatchedSettlement,
			CreatedAt:   createdAt,
		}, currenciestestutils.NewFiatCurrency(t, "USD"))
		require.ErrorContains(t, err, `settlement currency "EUR" must match credit currency "USD"`)
	})

	t.Run("legacy custom currency is projected as resolved manual cost basis", func(t *testing.T) {
		mapped, err := fromDBCostBasis(&db.ChargeCreditPurchase{
			SchemaLevel: int(creditpurchase.SchemaLevelLegacy),
			Settlement:  settlement,
			CreatedAt:   createdAt,
		}, currenciestestutils.NewCustomCurrency(t, "TOKENS", 2))
		require.NoError(t, err)
		require.Nil(t, mapped.CostBasisID)
		require.NotNil(t, mapped.CostBasis)
		require.NotNil(t, mapped.ResolvedCostBasis)
		require.Equal(t, createdAt, mapped.ResolvedCostBasis.ResolvedAt)
		require.Equal(t, 0.5, mapped.ResolvedCostBasis.CostBasis.InexactFloat64())

		intent, err := mapped.CostBasis.AsCustomCurrency()
		require.NoError(t, err)
		require.Equal(t, costbasis.ModeManual, intent.Kind())
	})

	t.Run("dedicated fiat does not fall back to settlement", func(t *testing.T) {
		_, err := fromDBCostBasis(&db.ChargeCreditPurchase{
			SchemaLevel:    int(creditpurchase.SchemaLevelCostBasis),
			Settlement:     settlement,
			SettlementType: &invoiceSettlementType,
			CreatedAt:      createdAt,
		}, currenciestestutils.NewFiatCurrency(t, "USD"))
		require.ErrorContains(t, err, "fiat cost basis is required")
	})

	t.Run("dedicated custom currency maps persisted state", func(t *testing.T) {
		customCurrency := currenciestestutils.NewCustomCurrency(t, "TOKENS", 2)
		customCurrency.Namespace = "ns"
		costBasisID := "01J00000000000000000000000"
		resolvedAt := createdAt.Add(time.Minute)

		mapped, err := fromDBCostBasis(&db.ChargeCreditPurchase{
			SchemaLevel:    int(creditpurchase.SchemaLevelCostBasis),
			Settlement:     settlement,
			SettlementType: &invoiceSettlementType,
			CostBasisID:    &costBasisID,
			Edges: db.ChargeCreditPurchaseEdges{
				CostBasis: &db.ChargeCreditPurchaseCostBasis{
					ID:                costBasisID,
					Namespace:         customCurrency.Namespace,
					Mode:              costbasis.ModeManual,
					FiatCurrency:      fiatCode,
					CurrencyID:        customCurrency.ID,
					ManualRate:        &rate,
					ResolvedCostBasis: &rate,
					ResolvedAt:        &resolvedAt,
					CreatedAt:         createdAt,
					UpdatedAt:         createdAt,
				},
			},
		}, customCurrency)
		require.NoError(t, err)
		require.NotNil(t, mapped.CostBasis)
		require.Equal(t, costBasisID, *mapped.CostBasisID)
		require.NotNil(t, mapped.ResolvedCostBasis)
		require.Equal(t, resolvedAt, mapped.ResolvedCostBasis.ResolvedAt)

		intent, err := mapped.CostBasis.AsCustomCurrency()
		require.NoError(t, err)
		require.Equal(t, costbasis.ModeManual, intent.Kind())
	})
}

func TestFromDBSettlement(t *testing.T) {
	t.Run("legacy maps the compatibility payload", func(t *testing.T) {
		currency := currencyx.FiatCode("USD")
		rate := alpacadecimal.NewFromFloat(0.5)
		initialStatus := creditpurchase.AuthorizedInitialPaymentSettlementStatus

		mapped, err := fromDBSettlement(&db.ChargeCreditPurchase{
			SchemaLevel: int(creditpurchase.SchemaLevelLegacy),
			Settlement: creditpurchase.PersistedSettlement{
				Type:          creditpurchase.SettlementTypeExternal,
				Currency:      &currency,
				CostBasis:     &rate,
				InitialStatus: &initialStatus,
			},
		})
		require.NoError(t, err)

		external, err := mapped.AsExternalSettlement()
		require.NoError(t, err)
		require.Equal(t, initialStatus, external.InitialStatus)
	})

	t.Run("dedicated mapping ignores the compatibility payload", func(t *testing.T) {
		settlementType := creditpurchase.SettlementTypeInvoice

		mapped, err := fromDBSettlement(&db.ChargeCreditPurchase{
			SchemaLevel:    int(creditpurchase.SchemaLevelCostBasis),
			SettlementType: &settlementType,
			Settlement:     creditpurchase.PersistedSettlement{},
		})
		require.NoError(t, err)
		require.Equal(t, creditpurchase.SettlementTypeInvoice, mapped.Type())
	})
}

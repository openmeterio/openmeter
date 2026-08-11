package adapter

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
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
	settlement := `{"type":"invoice","currency":"USD","costBasis":"0.5"}`

	t.Run("dedicated fiat does not fall back to settlement", func(t *testing.T) {
		_, err := fromDBCostBasis(&db.ChargeCreditPurchase{
			Settlement:     &settlement,
			SettlementType: &invoiceSettlementType,
			CreatedAt:      createdAt,
		}, currenciestestutils.NewFiatCurrency(t, "USD"))
		require.ErrorContains(t, err, "fiat cost basis is required")
	})

	t.Run("dedicated fiat materializes resolved state at charge creation", func(t *testing.T) {
		mapped, err := fromDBCostBasis(&db.ChargeCreditPurchase{
			SettlementType: &invoiceSettlementType,
			FiatCostBasis:  &rate,
			CreatedAt:      createdAt,
		}, currenciestestutils.NewFiatCurrency(t, "USD"))
		require.NoError(t, err)
		require.NotNil(t, mapped.CostBasis)
		require.Equal(t, &costbasis.State{
			CostBasis:  rate,
			ResolvedAt: createdAt,
		}, mapped.ResolvedCostBasis)
	})

	t.Run("dedicated custom currency maps persisted state", func(t *testing.T) {
		customCurrency := currenciestestutils.NewCustomCurrency(t, "TOKENS", 2)
		customCurrency.Namespace = "ns"
		costBasisID := "01J00000000000000000000000"
		resolvedAt := createdAt.Add(time.Minute)

		mapped, err := fromDBCostBasis(&db.ChargeCreditPurchase{
			Settlement:     &settlement,
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
		require.NotNil(t, mapped.ResolvedCostBasis)
		require.Equal(t, resolvedAt, mapped.ResolvedCostBasis.ResolvedAt)

		intent, err := mapped.CostBasis.AsCustomCurrency()
		require.NoError(t, err)
		require.Equal(t, costbasis.ModeManual, intent.Kind())
	})
}

func TestFromDBSettlement(t *testing.T) {
	t.Run("dedicated mapping ignores the compatibility payload", func(t *testing.T) {
		settlementType := creditpurchase.SettlementTypeInvoice
		settlement := `{}`

		mapped, err := fromDBSettlement(&db.ChargeCreditPurchase{
			SettlementType: &settlementType,
			Settlement:     &settlement,
		})
		require.NoError(t, err)
		require.Equal(t, creditpurchase.SettlementTypeInvoice, mapped.Type())
	})
}

func TestFromDBBaseRejectsUnsupportedSchemaLevel(t *testing.T) {
	_, err := fromDBBase(&db.ChargeCreditPurchase{SchemaLevel: 1}, meta.Charge{})
	require.ErrorContains(t, err, "unsupported credit purchase schema level: 1")
}

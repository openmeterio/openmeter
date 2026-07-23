package productcatalog

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidateCurrencyLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	currency := mustManagedCustomCurrency(t, "custom-currency-id", currencyx.Code("CREDITS"))
	currency.CostBasis = &[]currencies.CostBasis{}

	t.Run("archived custom currency is rejected", func(t *testing.T) {
		archived := currency.Clone()
		archived.DeletedAt = lo.ToPtr(now)

		require.ErrorIs(t, ValidateCurrency()(archived.Reference()), ErrCurrencyNotFound)
	})

	t.Run("scheduled custom currency deletion remains valid", func(t *testing.T) {
		scheduled := currency.Clone()
		scheduled.DeletedAt = lo.ToPtr(now.Add(time.Hour))

		require.NoError(t, ValidateCurrency()(scheduled.Reference()))
	})
}

func TestRateCardCurrencyRequiresPrice(t *testing.T) {
	custom := currencyx.Code("CREDITS")

	t.Run("currency without price is invalid", func(t *testing.T) {
		err := (RateCardMeta{Currency: currencyReferencePointer(custom)}).Validate()
		require.ErrorIs(t, err, ErrRateCardCurrencyRequiresPrice)
	})

	t.Run("currency with price is valid", func(t *testing.T) {
		err := (RateCardMeta{
			Currency: currencyReferencePointer(custom),
			Price: NewPriceFrom(FlatPrice{
				Amount:      decimal.NewFromInt(1),
				PaymentTerm: InAdvancePaymentTerm,
			}),
		}).Validate()
		require.NoError(t, err)
	})
}

func newCurrencyTestRateCard(key string, reference currencies.CurrencyReference) RateCard {
	return &FlatFeeRateCard{RateCardMeta: RateCardMeta{
		Key:      key,
		Name:     key,
		Currency: lo.ToPtr(reference),
		Price: NewPriceFrom(FlatPrice{
			Amount:      decimal.NewFromInt(1),
			PaymentTerm: InAdvancePaymentTerm,
		}),
	}}
}

func mustManagedCustomCurrency(t *testing.T, id string, code currencyx.Code) currencies.Currency {
	t.Helper()

	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(code).
		WithName(code.String()).
		Build()
	require.NoError(t, err)

	return currencies.Currency{
		NamespacedID: models.NamespacedID{ID: id},
		Currency:     currency,
	}
}

func mustFiatCurrencyReference(t *testing.T, code currencyx.Code) currencies.CurrencyReference {
	t.Helper()

	currency, err := currencies.NewFiatCurrency(code)
	require.NoError(t, err)

	return currency.Reference()
}

func currencyReferencePointer(code currencyx.Code) *currencies.CurrencyReference {
	reference := currencies.NewCurrencyReference(code)
	return &reference
}

package testutils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewManagedCurrency(t *testing.T, namespace, id string, code currencyx.Code) currencies.Currency {
	t.Helper()

	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(code).
		WithName(code.String()).
		Build()
	require.NoError(t, err)

	return currencies.Currency{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        id,
		},
		Currency: currency,
	}
}

func NewCreateCurrencyInput(namespace string, code currencyx.Code, name, symbol string) currencies.CreateCurrencyInput {
	return currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               code,
			Name:               name,
			Symbol:             symbol,
			Precision:          2,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	}
}

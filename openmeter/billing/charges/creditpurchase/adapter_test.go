package creditpurchase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestListChargesInputValidateCurrencyFormat(t *testing.T) {
	require.NoError(t, ListChargesInput{
		Namespace:  "namespace",
		Currencies: []currencyx.Code{"ACME_CREDITS"},
	}.Validate())
	require.Error(t, ListChargesInput{
		Namespace:  "namespace",
		Currencies: []currencyx.Code{"x"},
	}.Validate())
}

package creditgrant

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestListInputValidateCurrencyFormat(t *testing.T) {
	custom := currencyx.Code("ACME_CREDITS")
	malformed := currencyx.Code("x")

	require.NoError(t, ListInput{
		Namespace:  "namespace",
		CustomerID: "customer",
		Currency:   &custom,
	}.Validate())
	require.Error(t, ListInput{
		Namespace:  "namespace",
		CustomerID: "customer",
		Currency:   &malformed,
	}.Validate())
}

package ledger

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestAccountRouteParamsPreserveSource(t *testing.T) {
	source := currencyx.Code("USD")
	status := TransactionAuthorizationStatusOpen

	tests := []struct {
		name  string
		route Route
	}{
		{
			name: "customer fbo",
			route: CustomerFBORouteParams{
				Currency:       currencyx.Code("CREDITS"),
				Source:         &source,
				CreditPriority: DefaultCustomerFBOPriority,
			}.Route(),
		},
		{
			name: "customer receivable",
			route: CustomerReceivableRouteParams{
				Currency:                       currencyx.Code("CREDITS"),
				Source:                         &source,
				TransactionAuthorizationStatus: status,
			}.Route(),
		},
		{
			name: "customer accrued",
			route: CustomerAccruedRouteParams{
				Currency: currencyx.Code("CREDITS"),
				Source:   &source,
			}.Route(),
		},
		{
			name: "business",
			route: BusinessRouteParams{
				Currency: currencyx.Code("CREDITS"),
				Source:   &source,
			}.Route(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, &source, tt.route.Source)
		})
	}
}

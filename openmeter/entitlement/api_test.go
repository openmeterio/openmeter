package entitlement

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
)

func TestDeleteCustomerEntitlementInputValidate(t *testing.T) {
	valid := DeleteCustomerEntitlementInput{
		CustomerID:    customer.CustomerID{Namespace: "ns", ID: "01K4WAQ0J99ZZ0MD75HXR112H8"},
		EntitlementID: "01K4WAQ0J99ZZ0MD75HXR112H9",
	}

	require.NoError(t, valid.Validate())

	t.Run("requires customer id", func(t *testing.T) {
		input := valid
		input.CustomerID = customer.CustomerID{Namespace: "ns"}

		require.ErrorContains(t, input.Validate(), "customer ID")
	})

	t.Run("requires entitlement id", func(t *testing.T) {
		input := valid
		input.EntitlementID = ""

		require.ErrorContains(t, input.Validate(), "entitlement ID is required")
	})
}

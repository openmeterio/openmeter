package entitlement

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestListCustomerEntitlementGrantsInputValidate(t *testing.T) {
	valid := ListCustomerEntitlementGrantsInput{
		CustomerID:    customer.CustomerID{Namespace: "ns", ID: "01K4WAQ0J99ZZ0MD75HXR112H8"},
		EntitlementID: "01K4WAQ0J99ZZ0MD75HXR112H9",
		OrderBy:       grant.OrderByEffectiveAt,
		Page:          pagination.NewPage(1, 20),
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

	t.Run("allows an unset order by", func(t *testing.T) {
		input := valid
		input.OrderBy = ""

		require.NoError(t, input.Validate())
	})

	t.Run("rejects an unknown order by", func(t *testing.T) {
		input := valid
		input.OrderBy = "amount"

		require.ErrorContains(t, input.Validate(), "invalid order by: amount")
	})

	t.Run("requires a page", func(t *testing.T) {
		input := valid
		input.Page = pagination.Page{}

		require.ErrorContains(t, input.Validate(), "page is required")
	})

	t.Run("rejects an invalid page", func(t *testing.T) {
		input := valid
		input.Page = pagination.NewPage(0, 20)

		require.ErrorContains(t, input.Validate(), "page:")
	})
}

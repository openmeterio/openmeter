package entitlement

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

func TestCreateCustomerEntitlementInputValidate(t *testing.T) {
	valid := CreateCustomerEntitlementInput{
		CustomerID: customer.CustomerID{Namespace: "ns", ID: "01K4WAQ0J99ZZ0MD75HXR112H8"},
		Entitlement: CreateEntitlementInputs{
			FeatureID:       lo.ToPtr("01K4WAQ0J99ZZ0MD75HXR112H9"),
			EntitlementType: EntitlementTypeMetered,
		},
	}

	require.NoError(t, valid.Validate())

	t.Run("requires customer id", func(t *testing.T) {
		input := valid
		input.CustomerID = customer.CustomerID{Namespace: "ns"}

		require.ErrorContains(t, input.Validate(), "customer ID")
	})

	t.Run("requires feature", func(t *testing.T) {
		input := valid
		input.Entitlement.FeatureID = nil

		require.ErrorContains(t, input.Validate(), "feature is required")
	})

	t.Run("rejects issue after reset combined with grants", func(t *testing.T) {
		input := valid
		input.Entitlement.IssueAfterReset = lo.ToPtr(100.0)
		input.Grants = []CreateEntitlementGrantInputs{{CreateGrantInput: credit.CreateGrantInput{Amount: 10}}}

		require.ErrorContains(t, input.Validate(), "cannot be used together")
	})
}

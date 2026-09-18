package entitlement

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
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

func TestGetCustomerEntitlementInputValidate(t *testing.T) {
	valid := GetCustomerEntitlementInput{
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

func TestListCustomerEntitlementsInputValidate(t *testing.T) {
	valid := ListCustomerEntitlementsInput{
		CustomerID: customer.CustomerID{Namespace: "ns", ID: "01K4WAQ0J99ZZ0MD75HXR112H8"},
		FeatureID:  &filter.FilterULID{Eq: lo.ToPtr("01K4WAQ0J99ZZ0MD75HXR112H9")},
		FeatureKey: &filter.FilterString{In: &[]string{"a", "b"}},
		Type:       &filter.FilterString{Eq: lo.ToPtr(string(EntitlementTypeMetered))},
		OrderBy:    ListEntitlementsOrderByUpdatedAt,
		Page:       pagination.NewPage(1, 10),
	}

	require.NoError(t, valid.Validate())

	t.Run("allows an unset page", func(t *testing.T) {
		input := valid
		input.Page = pagination.Page{}

		require.NoError(t, input.Validate())
	})

	t.Run("requires customer id", func(t *testing.T) {
		input := valid
		input.CustomerID = customer.CustomerID{Namespace: "ns"}

		require.ErrorContains(t, input.Validate(), "customer ID")
	})

	t.Run("rejects an unknown entitlement type", func(t *testing.T) {
		input := valid
		input.Type = &filter.FilterString{In: &[]string{"metered", "unknown"}}

		require.ErrorContains(t, input.Validate(), "invalid entitlement type: unknown")
	})

	t.Run("rejects a malformed feature id filter", func(t *testing.T) {
		input := valid
		input.FeatureID = &filter.FilterULID{Eq: lo.ToPtr("not-a-ulid")}

		require.ErrorContains(t, input.Validate(), "feature ID filter")
	})

	t.Run("rejects multiple operators on one filter", func(t *testing.T) {
		input := valid
		input.FeatureKey = &filter.FilterString{Eq: lo.ToPtr("a"), In: &[]string{"b"}}

		require.ErrorContains(t, input.Validate(), "feature key filter")
	})

	t.Run("rejects an unknown order by", func(t *testing.T) {
		input := valid
		input.OrderBy = "feature_key"

		require.ErrorContains(t, input.Validate(), "invalid order by: feature_key")
	})

	t.Run("rejects an invalid page", func(t *testing.T) {
		input := valid
		input.Page = pagination.NewPage(0, 10)

		require.ErrorContains(t, input.Validate(), "page")
	})
}

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

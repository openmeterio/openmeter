package httpdriver

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/customer/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCustomerToAPISubscriptionExpansion(t *testing.T) {
	c := customer.Customer{
		Key:      lo.ToPtr("customer-1"),
		Metadata: lo.ToPtr(models.Metadata{"team": "billing"}),
	}
	base := apiconvert.ToAPICustomer(c)
	require.Nil(t, base.CurrentSubscriptionId)
	require.Nil(t, base.Subscriptions)

	got, err := CustomerToAPI(c, nil, nil)
	require.NoError(t, err)
	require.Equal(t, base, got)

	subs := []subscription.Subscription{{NamespacedID: models.NamespacedID{ID: "sub-1"}}}
	got, err = CustomerToAPI(c, subs, nil)
	require.NoError(t, err)
	require.Equal(t, lo.ToPtr("sub-1"), got.CurrentSubscriptionId)
	require.Nil(t, got.Subscriptions)

	got, err = CustomerToAPI(c, subs, customer.Expands{customer.ExpandSubscriptions})
	require.NoError(t, err)
	require.Equal(t, lo.ToPtr("sub-1"), got.CurrentSubscriptionId)
	require.NotNil(t, got.Subscriptions)
	require.Len(t, *got.Subscriptions, 1)
	require.Equal(t, "sub-1", (*got.Subscriptions)[0].Id)
}

package customerservice_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customertestutils "github.com/openmeterio/openmeter/openmeter/customer/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func Test_ListCustomersUpdatedAt(t *testing.T) {
	env := customertestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := customertestutils.NewTestNamespace(t)
	ctx := t.Context()

	createCustomer := func(key string) *customer.Customer {
		cus, err := env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:              lo.ToPtr(key),
				Name:             key,
				UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{key}},
			},
		})
		require.NoError(t, err)

		return cus
	}

	list := func(updatedAt filter.FilterTime) pagination.Result[customer.Customer] {
		result, err := env.CustomerService.ListCustomers(ctx, customer.ListCustomersInput{
			Namespace: namespace,
			Page:      pagination.NewPage(1, 1),
			UpdatedAt: &updatedAt,
		})
		require.NoError(t, err)

		return result
	}

	// given: a customer written before a sync checkpoint and one written after it.
	checkpoint := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	clock.FreezeTime(checkpoint.Add(-time.Hour))
	defer clock.UnFreeze()
	before := createCustomer("before-checkpoint")

	clock.FreezeTime(checkpoint)
	atCheckpoint := createCustomer("at-checkpoint")

	// when/then: gte includes the boundary, gt excludes it, and lt selects the
	// earlier customer; page totals count only the filtered rows.
	gte := list(filter.FilterTime{Gte: &checkpoint})
	require.Equal(t, []string{atCheckpoint.ID}, lo.Map(gte.Items, func(c customer.Customer, _ int) string { return c.ID }))
	require.Equal(t, 1, gte.TotalCount)

	require.Empty(t, list(filter.FilterTime{Gt: &checkpoint}).Items)

	lt := list(filter.FilterTime{Lt: &checkpoint})
	require.Equal(t, []string{before.ID}, lo.Map(lt.Items, func(c customer.Customer, _ int) string { return c.ID }))
	require.Equal(t, 1, lt.TotalCount)
}

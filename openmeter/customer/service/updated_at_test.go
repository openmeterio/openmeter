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

	// given: two customers written before a sync checkpoint.
	checkpoint := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	clock.FreezeTime(checkpoint.Add(-time.Hour))
	defer clock.UnFreeze()
	createCustomer("unchanged")
	changed := createCustomer("changed")

	// when: one of them is updated at the checkpoint.
	clock.FreezeTime(checkpoint)
	defer clock.UnFreeze()

	mutate := changed.AsCustomerMutate()
	mutate.Name = "changed-renamed"
	_, err := env.CustomerService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID:     changed.GetID(),
		CustomerMutate: mutate,
	})
	require.NoError(t, err)

	// then: only the updated customer matches. Its created_at is before the
	// checkpoint, so this can only match on updated_at.
	result, err := env.CustomerService.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: namespace,
		Page:      pagination.NewPage(1, 10),
		UpdatedAt: &filter.FilterTime{Gte: &checkpoint},
	})
	require.NoError(t, err)
	require.Equal(t, []string{changed.ID}, lo.Map(result.Items, func(c customer.Customer, _ int) string { return c.ID }))
}

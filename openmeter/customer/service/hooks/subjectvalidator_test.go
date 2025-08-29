package hooks

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
)

func Test_SubjectValidatorHook(t *testing.T) {
	// Setup test environment
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := NewTestNamespace(t)

	ctx := t.Context()

	hook, err := NewSubjectValidatorHook(SubjectValidatorHookConfig{
		Customer: env.CustomerService,
		Logger:   env.Logger,
	})
	require.NoError(t, err, "creating subject validator hook should not fail")
	require.NotNilf(t, hook, "subject validator hook must not be nil")

	env.SubjectService.RegisterHooks(hook)

	t.Run("Create", func(t *testing.T) {
		sub, err := env.SubjectService.Create(ctx, subject.CreateInput{
			Namespace:   namespace,
			Key:         "acme-inc",
			DisplayName: lo.ToPtr("ACME Inc."),
		})
		require.NoError(t, err, "creating subject should not fail")
		assert.NotNilf(t, sub, "subject must not be nil")

		cus, err := env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr("acme-inc"),
				Name: "ACME Inc.",
				UsageAttribution: customer.CustomerUsageAttribution{
					SubjectKeys: []string{
						sub.Key,
					},
				},
			},
		})
		require.NoError(t, err, "creating customer should not fail")
		assert.NotNilf(t, cus, "customer must not be nil")

		sub2, err := env.SubjectService.Create(ctx, subject.CreateInput{
			Namespace:   namespace,
			Key:         "example-inc",
			DisplayName: lo.ToPtr("Example Inc."),
		})
		require.NoError(t, err, "creating subject should not fail")
		assert.NotNilf(t, sub2, "subject must not be nil")

		t.Run("Delete", func(t *testing.T) {
			t.Run("DeleteSubjectWithCustomer", func(t *testing.T) {
				err = env.SubjectService.Delete(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        sub.Id,
				})
				require.ErrorAsf(t, err, new(*models.GenericValidationError), "error must be validation error")

				t.Run("UpdateCustomer", func(t *testing.T) {
					cus, err = env.CustomerService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
						CustomerID: customer.CustomerID{Namespace: namespace, ID: cus.ID},
						CustomerMutate: customer.CustomerMutate{
							Name: cus.Name,
							UsageAttribution: customer.CustomerUsageAttribution{
								SubjectKeys: []string{},
							},
						},
					})
					require.NoError(t, err, "updating customer should not fail")
					assert.NotNilf(t, cus, "customer must not be nil")

					t.Run("DeleteSubject", func(t *testing.T) {
						err = env.SubjectService.Delete(ctx, models.NamespacedID{
							Namespace: namespace,
							ID:        sub.Id,
						})
						require.NoErrorf(t, err, "deleting subject should not fail")
					})
				})
			})

			t.Run("DeleteSubjectWithoutCustomer", func(t *testing.T) {
				err = env.SubjectService.Delete(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        sub2.Id,
				})
				require.NoErrorf(t, err, "deleting subject should not fail")
			})
		})
	})
}

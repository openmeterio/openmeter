package customerservice_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customertestutils "github.com/openmeterio/openmeter/openmeter/customer/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func Test_CustomerService(t *testing.T) {
	// Setup test environment
	env := customertestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := customertestutils.NewTestNamespace(t)

	ctx := t.Context()

	t.Run("Customer", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			cus, err := env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
				Namespace: namespace,
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr("acme-inc"),
					Name: "ACME Inc.",
					UsageAttribution: &customer.CustomerUsageAttribution{
						SubjectKeys: []string{
							"acme-inc",
						},
					},
				},
			})
			require.NoError(t, err, "creating customer should not fail")
			assert.NotNilf(t, cus, "customer must not be nil")
			assert.Equalf(t, cus.Key, cus.Key, "customer key must match")
			assert.Equalf(t, cus.Name, cus.Name, "customer name must match")
			assert.ElementsMatchf(t, cus.UsageAttribution.SubjectKeys, []string{"acme-inc"}, "customer usage attribution must match")

			t.Run("Get", func(t *testing.T) {
				t.Run("ByID", func(t *testing.T) {
					cusByID, err := env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
						CustomerID: &customer.CustomerID{
							Namespace: cus.Namespace,
							ID:        cus.ID,
						},
					})
					require.NoError(t, err, "getting customer by id should not fail")
					assert.NotNilf(t, cusByID, "customer must not be nil")
					assert.Equal(t, cus.ID, cusByID.ID, "customer id must match")
				})

				t.Run("ByKey", func(t *testing.T) {
					cusByKey, err := env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
						CustomerKey: &customer.CustomerKey{
							Namespace: cus.Namespace,
							Key:       lo.FromPtr(cus.Key),
						},
					})
					require.NoError(t, err, "getting customer by key should not fail")
					assert.NotNilf(t, cusByKey, "customer must not be nil")
					assert.Equal(t, cus.ID, cusByKey.ID, "customer id must match")
				})

				t.Run("ByIDDOrKey", func(t *testing.T) {
					cusByIDOrKey, err := env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
						CustomerIDOrKey: &customer.CustomerIDOrKey{
							Namespace: cus.Namespace,
							IDOrKey:   cus.ID,
						},
					})
					require.NoError(t, err, "getting customer by id or key should not fail")
					assert.NotNilf(t, cusByIDOrKey, "customer must not be nil")
					assert.Equal(t, cus.ID, cusByIDOrKey.ID, "customer id must match")
				})

				t.Run("ByUsageAttribution", func(t *testing.T) {
					cusByUsage, err := env.CustomerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
						Namespace: cus.Namespace,
						Key:       cus.UsageAttribution.SubjectKeys[0],
					})
					require.NoError(t, err, "getting customer usage attribution should not fail")
					assert.NotNilf(t, cusByUsage, "customer must not be nil")
					assert.Equal(t, cus.ID, cusByUsage.ID, "customer id must match")
				})
			})

			t.Run("List", func(t *testing.T) {
				list, err := env.CustomerService.ListCustomers(ctx, customer.ListCustomersInput{
					Namespace: namespace,
				})
				require.NoError(t, err, "listing customers should not fail")
				assert.NotNilf(t, list, "customer list must not be nil")
				assert.NotEmptyf(t, list.Items, "customer list must not be empty")
			})

			subjectKeys := []string{"acme-inc", "acme-llc"}

			t.Run("Update", func(t *testing.T) {
				updatedCus, err := env.CustomerService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
					CustomerID: customer.CustomerID{
						Namespace: cus.Namespace,
						ID:        cus.ID,
					},
					CustomerMutate: customer.CustomerMutate{
						Key:  cus.Key,
						Name: cus.Name,
						UsageAttribution: &customer.CustomerUsageAttribution{
							SubjectKeys: subjectKeys,
						},
					},
				})
				require.NoError(t, err, "updating customer should not fail")
				assert.NotNilf(t, updatedCus, "customer must not be nil")
				assert.Equalf(t, updatedCus.Key, cus.Key, "customer key must match")
				assert.Equalf(t, updatedCus.Name, cus.Name, "customer name must match")
				assert.ElementsMatchf(t, updatedCus.UsageAttribution.SubjectKeys, subjectKeys, "customer usage attribution must match")

				t.Run("ByUsageAttribution", func(t *testing.T) {
					cusByUsage, err := env.CustomerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
						Namespace: cus.Namespace,
						Key:       subjectKeys[1],
					})
					require.NoError(t, err, "getting customer usage attribution should not fail")
					assert.NotNilf(t, cusByUsage, "customer must not be nil")
					assert.Equal(t, updatedCus.ID, cusByUsage.ID, "customer id must match")
				})
			})

			t.Run("Delete", func(t *testing.T) {
				err = env.CustomerService.DeleteCustomer(ctx, customer.DeleteCustomerInput{
					Namespace: cus.Namespace,
					ID:        cus.ID,
				})
				require.NoError(t, err, "deleting customer should not fail")

				t.Run("Get", func(t *testing.T) {
					t.Run("ByID", func(t *testing.T) {
						cusByID, err := env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
							CustomerID: &customer.CustomerID{
								Namespace: cus.Namespace,
								ID:        cus.ID,
							},
						})
						require.NoError(t, err, "getting customer by id should not fail")
						assert.NotNilf(t, cusByID, "customer must not be nil")
						assert.Equal(t, cus.ID, cusByID.ID, "customer id must match")
						assert.NotNilf(t, cusByID.DeletedAt, "customer must be deleted")
						assert.ElementsMatchf(t, cusByID.UsageAttribution.SubjectKeys, subjectKeys, "customer usage attribution must match")
					})

					t.Run("ByKey", func(t *testing.T) {
						cusByKey, err := env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
							CustomerKey: &customer.CustomerKey{
								Namespace: cus.Namespace,
								Key:       lo.FromPtr(cus.Key),
							},
						})

						var notFoundErr *models.GenericNotFoundError

						assert.ErrorAsf(t, err, &notFoundErr, "error must be not found error")
						assert.Nilf(t, cusByKey, "customer must be nil")
					})

					t.Run("ByIDOrKey", func(t *testing.T) {
						t.Run("ByID", func(t *testing.T) {
							cusByID, err := env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
								CustomerID: &customer.CustomerID{
									Namespace: cus.Namespace,
									ID:        cus.ID,
								},
							})
							require.NoError(t, err, "getting customer by id should not fail")
							assert.NotNilf(t, cusByID, "customer must not be nil")
							assert.Equal(t, cus.ID, cusByID.ID, "customer id must match")
							assert.NotNilf(t, cusByID.DeletedAt, "customer must be deleted")
							assert.ElementsMatchf(t, cusByID.UsageAttribution.SubjectKeys, subjectKeys, "customer usage attribution must match")
						})

						t.Run("ByKey", func(t *testing.T) {
							cusByKey, err := env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
								CustomerKey: &customer.CustomerKey{
									Namespace: cus.Namespace,
									Key:       lo.FromPtr(cus.Key),
								},
							})

							var notFoundErr *models.GenericNotFoundError

							assert.ErrorAsf(t, err, &notFoundErr, "error must be not found error")
							assert.Nilf(t, cusByKey, "customer must be nil")
						})
					})
				})

				t.Run("List", func(t *testing.T) {
					list, err := env.CustomerService.ListCustomers(ctx, customer.ListCustomersInput{
						Namespace:      namespace,
						IncludeDeleted: true,
					})
					require.NoError(t, err, "listing customers including deleted should not fail")
					require.NotNilf(t, list, "customer list must not be nil")
					require.NotEmptyf(t, list.Items, "customer list must not be empty")
					assert.Equal(t, cus.ID, list.Items[0].ID, "customer id must match")
					assert.NotNilf(t, list.Items[0].DeletedAt, "customer must be deleted")
					assert.ElementsMatchf(t, list.Items[0].UsageAttribution.SubjectKeys, subjectKeys, "customer usage attribution must match")
				})
			})
		})
	})
}

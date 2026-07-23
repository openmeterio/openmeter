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

// Test_GetCustomersByUsageAttribution protects the bulk method's contract with the same rigor as
// Test_GetCustomerByUsageAttribution does the single-key one: each input key maps to its customer
// with key-over-subject precedence, and unmatched keys are present with a nil value (no error). The predicate
// itself is covered by the adapter's TestGetCustomersByUsageAttribution, and the fully symmetric
// {K1,K2} cross-collision by Test_resolveCustomersByKeyWithPrecedence — that state cannot be built
// through the service here, because CreateCustomer rejects a customer whose key overlaps another
// customer's subject key (see the overlap guard in the customer adapter).
func Test_GetCustomersByUsageAttribution(t *testing.T) {
	env := customertestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := customertestutils.NewTestNamespace(t)
	ctx := t.Context()

	create := func(t *testing.T, key string, subjectKeys ...string) *customer.Customer {
		t.Helper()
		cus, err := env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:              lo.ToPtr(key),
				Name:             key,
				UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: subjectKeys},
			},
		})
		require.NoError(t, err)
		return cus
	}

	t.Run("ResolvesByKeyAndSubject", func(t *testing.T) {
		// given:
		// - customer A matched by its own key, customer B matched by one of its subject keys
		// then:
		// - each input key maps to the correct distinct customer
		a := create(t, "a-key", "a-subject")
		b := create(t, "b-key", "b-subject")

		got, err := env.CustomerService.GetCustomersByUsageAttribution(ctx, customer.GetCustomersByUsageAttributionInput{
			Namespace: namespace,
			Keys:      []string{"a-key", "b-subject"},
		})
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, a.ID, got["a-key"].ID)
		assert.Equal(t, b.ID, got["b-subject"].ID)
	})

	t.Run("KeyOverSubjectPrecedence", func(t *testing.T) {
		// given:
		// - "shared" is customer P's own key AND customer Q's subject key
		// then:
		// - the key owner (P) wins, resolved in the service
		p := create(t, "shared", "p-subject")
		_ = create(t, "q-key", "shared")

		got, err := env.CustomerService.GetCustomersByUsageAttribution(ctx, customer.GetCustomersByUsageAttributionInput{
			Namespace: namespace,
			Keys:      []string{"shared"},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, p.ID, got["shared"].ID, "a direct customer-key match must win over a subject-key match")
	})

	t.Run("UnmatchedKeysNil", func(t *testing.T) {
		// given:
		// - a mix of a known key and an unknown one
		// then:
		// - every input key is present in the map; the unknown key has a nil value (unlike the
		//   single-key method, the bulk method does NOT return a not-found error)
		known := create(t, "known-key")

		got, err := env.CustomerService.GetCustomersByUsageAttribution(ctx, customer.GetCustomersByUsageAttributionInput{
			Namespace: namespace,
			Keys:      []string{"known-key", "totally-unknown"},
		})
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.NotNil(t, got["known-key"])
		assert.Equal(t, known.ID, got["known-key"].ID)
		unknown, ok := got["totally-unknown"]
		assert.True(t, ok, "every input key must be present in the result map")
		assert.Nil(t, unknown, "unmatched key must have a nil value")
	})

	t.Run("EmptyKeysFailsValidation", func(t *testing.T) {
		// given:
		// - an empty key set
		// then:
		// - the service rejects it with a validation error before any DB query is built
		_, err := env.CustomerService.GetCustomersByUsageAttribution(ctx, customer.GetCustomersByUsageAttributionInput{
			Namespace: namespace,
			Keys:      nil,
		})
		require.Error(t, err, "empty key set must fail validation")
		assert.True(t, models.IsGenericValidationError(err), "error must be a validation error")
	})
}

// Test_GetCustomerByUsageAttribution covers the single-key wiring that Test_CustomerService's
// happy-path lookups don't reach: the key-over-subject precedence now resolved in the service (not
// in SQL), the not-found mapping, and input validation. The predicate itself is covered by the
// adapter's TestGetCustomersByUsageAttribution.
func Test_GetCustomerByUsageAttribution(t *testing.T) {
	env := customertestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := customertestutils.NewTestNamespace(t)
	ctx := t.Context()

	t.Run("KeyOverSubjectPrecedence", func(t *testing.T) {
		// given:
		// - customer A owns the key "shared-key"
		// - customer B carries "shared-key" as one of its subject keys
		// when:
		// - resolving "shared-key" by usage attribution
		// then:
		// - the direct key owner (A) wins over the subject-key match (B), resolved in the service
		keyOwner, err := env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:              lo.ToPtr("shared-key"),
				Name:             "Key Owner",
				UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{"key-owner-subject"}},
			},
		})
		require.NoError(t, err)

		_, err = env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:              lo.ToPtr("subject-owner"),
				Name:             "Subject Owner",
				UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{"shared-key"}},
			},
		})
		require.NoError(t, err)

		got, err := env.CustomerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
			Namespace: namespace,
			Key:       "shared-key",
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, keyOwner.ID, got.ID, "a direct customer-key match must win over a subject-key match")
	})

	t.Run("UnknownKeyReturnsNotFound", func(t *testing.T) {
		_, err := env.CustomerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
			Namespace: namespace,
			Key:       "no-such-key",
		})
		require.Error(t, err)
		assert.True(t, models.IsGenericNotFoundError(err), "unmatched key must map to a not-found error")
	})

	t.Run("EmptyKeyFailsValidation", func(t *testing.T) {
		_, err := env.CustomerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
			Namespace: namespace,
			Key:       "",
		})
		require.Error(t, err)
		assert.True(t, models.IsGenericValidationError(err), "empty key must fail validation")
	})
}

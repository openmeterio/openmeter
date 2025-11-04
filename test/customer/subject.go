package customer

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *CustomerHandlerTestSuite) TestSubjectDeletion(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	t.Run("Should allow deletion of a dangling subject (dangling after removing from customer usage attribution)", func(t *testing.T) {
		// Let's create a customer with a subject
		cust, err := s.Env.Customer().CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr("customer-1"),
				Name: "Customer 1",
				UsageAttribution: customer.CustomerUsageAttribution{
					SubjectKeys: []string{"customer-1-subject-1"},
				},
			},
		})

		require.NoError(t, err, "Creating customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		// Let's make the subject dangling by removing it from the customer usage attribution
		cust, err = s.Env.Customer().UpdateCustomer(ctx, customer.UpdateCustomerInput{
			CustomerID: customer.CustomerID{
				Namespace: s.namespace,
				ID:        cust.ID,
			},
			CustomerMutate: func() customer.CustomerMutate {
				mut := cust.AsCustomerMutate()
				mut.UsageAttribution.SubjectKeys = []string{}

				return mut
			}(),
		})
		require.NoError(t, err, "Updating customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		// Let's fetch the subject
		subj, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{
			Namespace: s.namespace,
			Key:       "customer-1-subject-1",
		})
		require.NoError(t, err, "Getting subject must not return error")
		require.NotNil(t, subj, "Subject must not be nil")

		// Let's delete the subject
		err = s.Env.Subject().Delete(ctx, models.NamespacedID{
			Namespace: s.namespace,
			ID:        subj.Id,
		})
		require.NoError(t, err, "Deleting subject must not return error")
	})

	t.Run("Should remove from customer usage attribution if not dangling", func(t *testing.T) {
		// Let's create a customer with a subject
		cust, err := s.Env.Customer().CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr("customer-2"),
				Name: "Customer 2",
				UsageAttribution: customer.CustomerUsageAttribution{
					SubjectKeys: []string{"customer-2-subject-1"},
				},
			},
		})

		require.NoError(t, err, "Creating customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		// Let's fetch the subject
		subj, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{
			Namespace: s.namespace,
			Key:       "customer-2-subject-1",
		})
		require.NoError(t, err, "Getting subject must not return error")
		require.NotNil(t, subj, "Subject must not be nil")

		// Let's delete the subject
		err = s.Env.Subject().Delete(ctx, models.NamespacedID{
			Namespace: s.namespace,
			ID:        subj.Id,
		})
		require.NoError(t, err, "Deleting subject must not return error")

		// Let's fetch the customer
		cust, err = s.Env.Customer().GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: s.namespace,
				ID:        cust.ID,
			},
		})
		require.NoError(t, err, "Getting customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		require.Equal(t, []string{}, cust.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must be empty")
	})

	t.Run("Should NOT error if customer WITH entitlements has no more subjects after deletion", func(t *testing.T) {
		// Let's create a customer with a subject
		cust, err := s.Env.Customer().CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr("customer-3"),
				Name: "Customer 3",
				UsageAttribution: customer.CustomerUsageAttribution{
					SubjectKeys: []string{"customer-3-subject-1"},
				},
			},
		})

		require.NoError(t, err, "Creating customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")

		// Let's create a feature
		feature, err := s.Env.Feature().CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: s.namespace,
			Key:       "test-feature",
			Name:      "Test Feature",
		})
		require.NoError(t, err, "Creating feature must not return error")
		require.NotNil(t, feature, "Feature must not be nil")

		// Let's create an entitlement for the customer
		entitlement, err := s.Env.Entitlement().CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
			Namespace:        s.namespace,
			FeatureID:        lo.ToPtr(feature.ID),
			EntitlementType:  entitlement.EntitlementTypeBoolean,
			UsageAttribution: cust.GetUsageAttribution(),
		}, nil)
		require.NoError(t, err, "Creating entitlement must not return error")
		require.NotNil(t, entitlement, "Entitlement must not be nil")

		// Let's fetch the subject
		subj, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{
			Namespace: s.namespace,
			Key:       "customer-3-subject-1",
		})
		require.NoError(t, err, "Getting subject must not return error")
		require.NotNil(t, subj, "Subject must not be nil")

		// Let's delete the subject
		err = s.Env.Subject().Delete(ctx, models.NamespacedID{
			Namespace: s.namespace,
			ID:        subj.Id,
		})
		require.NoError(t, err, "Deleting subject must not return error")

		// Let's assert that the customer has no more subjects
		cust, err = s.Env.Customer().GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: s.namespace,
				ID:        cust.ID,
			},
		})
		require.NoError(t, err, "Getting customer must not return error")
		require.NotNil(t, cust, "Customer must not be nil")
		require.Equal(t, []string{}, cust.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must be empty")
	})
}

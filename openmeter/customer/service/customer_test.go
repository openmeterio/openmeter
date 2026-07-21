package customerservice

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

func Test_resolveCustomersByKeyWithPrecedence(t *testing.T) {
	customerA := customer.Customer{
		ManagedResource: models.ManagedResource{ID: "customer-a"},
		Key:             lo.ToPtr("key-a"),
		UsageAttribution: &customer.CustomerUsageAttribution{
			SubjectKeys: []string{"subject-a"},
		},
	}

	t.Run("MatchByKey", func(t *testing.T) {
		resolved := resolveCustomersByKeyWithPrecedence([]customer.Customer{customerA}, []string{"key-a"})

		assert.Equal(t, map[string]*customer.Customer{"key-a": &customerA}, resolved)
	})

	t.Run("MatchBySubject", func(t *testing.T) {
		resolved := resolveCustomersByKeyWithPrecedence([]customer.Customer{customerA}, []string{"subject-a"})

		assert.Equal(t, map[string]*customer.Customer{"subject-a": &customerA}, resolved)
	})

	t.Run("UnmatchedKeyIsNil", func(t *testing.T) {
		resolved := resolveCustomersByKeyWithPrecedence([]customer.Customer{customerA}, []string{"no-such-key"})

		assert.Equal(t, map[string]*customer.Customer{"no-such-key": nil}, resolved)
	})

	t.Run("KeyOwnerTakesPrecedenceOverDistinctSubjectOwner", func(t *testing.T) {
		// "shared" is customerA's own key AND customerB's subject key: the key-owner (A) must win.
		sharedKeyCustomer := customer.Customer{
			ManagedResource: models.ManagedResource{ID: "customer-a"},
			Key:             lo.ToPtr("shared"),
		}
		sharedSubjectCustomer := customer.Customer{
			ManagedResource: models.ManagedResource{ID: "customer-b"},
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"shared"},
			},
		}

		resolved := resolveCustomersByKeyWithPrecedence(
			[]customer.Customer{sharedKeyCustomer, sharedSubjectCustomer},
			[]string{"shared"},
		)

		assert.Equal(t, map[string]*customer.Customer{"shared": &sharedKeyCustomer}, resolved)
	})

	t.Run("SameCustomerMatchedByOwnKeyAndSubjectKey", func(t *testing.T) {
		selfMatched := customer.Customer{
			ManagedResource: models.ManagedResource{ID: "customer-a"},
			Key:             lo.ToPtr("dual"),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"dual"},
			},
		}

		resolved := resolveCustomersByKeyWithPrecedence([]customer.Customer{selfMatched}, []string{"dual"})

		assert.Equal(t, map[string]*customer.Customer{"dual": &selfMatched}, resolved)
	})

	t.Run("CrossCollisionResolvesEachKeyToItsOwnKeyOwner", func(t *testing.T) {
		// customerA's key is customerB's subject key and vice versa: no single ordering of
		// [A, B] can satisfy both keys under a naive single-pass first-match-wins map, but
		// resolving key-owner and subject-owner independently per key does.
		crossA := customer.Customer{
			ManagedResource: models.ManagedResource{ID: "customer-a"},
			Key:             lo.ToPtr("key-1"),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"key-2"},
			},
		}
		crossB := customer.Customer{
			ManagedResource: models.ManagedResource{ID: "customer-b"},
			Key:             lo.ToPtr("key-2"),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"key-1"},
			},
		}

		resolved := resolveCustomersByKeyWithPrecedence(
			[]customer.Customer{crossA, crossB},
			[]string{"key-1", "key-2"},
		)

		assert.Equal(t, map[string]*customer.Customer{
			"key-1": &crossA,
			"key-2": &crossB,
		}, resolved)
	})
}

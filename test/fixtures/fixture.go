// Fixtures can be used for testing to dynamically create entities that we need in our tests.
package fixtures

import (
	"github.com/go-faker/faker/v4"
	"github.com/samber/do"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type InstanceProvider[T any] func(i *do.Injector) (T, error)

// FixtureDeps are actual services (or even just HTTP calls) that manage actually creating the entities
type FixtureDeps interface {
	CreateCustomer(customer customer.CreateCustomerInput) (customer.Customer, error)
}

var _ FixtureDeps = NoopDeps{}

type NoopDeps struct{}

func (NoopDeps) CreateCustomer(cust customer.CreateCustomerInput) (customer.Customer, error) {
	return customer.Customer{
		Key:              cust.Key,
		UsageAttribution: cust.UsageAttribution,
		PrimaryEmail:     cust.PrimaryEmail,
		ManagedResource: models.ManagedResource{
			Name:        cust.Name,
			Description: cust.Description,
		},
	}, nil
}

// Namesapce

const NamespaceFixtureName = "namespace"

type NamespaceFixture struct{}

func (f NamespaceFixture) Register(i *do.Injector) {
	do.ProvideNamed(i, NamespaceFixtureName, f.GetProvider())
}

func (f NamespaceFixture) GetProvider() do.Provider[string] {
	return func(i *do.Injector) (string, error) {
		return "default", nil
	}
}

// Customer

const (
	CustomerFixtureName         = "customer"
	CustomerFixtureInstanceName = "customer-instance"
)

type CustomerFixture struct {
	Deps FixtureDeps
}

func (f CustomerFixture) Register(i *do.Injector) {
	do.ProvideNamed(i, CustomerFixtureName, f.GetProvider())
	do.ProvideNamed(i, CustomerFixtureInstanceName, f.GetInstanceProvider())
}

func (f CustomerFixture) GetProvider() do.Provider[customer.Customer] {
	return func(i *do.Injector) (customer.Customer, error) {
		namespace, err := do.InvokeNamed[string](i, NamespaceFixtureName)
		if err != nil {
			return customer.Customer{}, err
		}

		inp := customer.CreateCustomerInput{
			Namespace: namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:          lo.ToPtr(RandKey()),
				Name:         faker.Name(),
				PrimaryEmail: lo.ToPtr(faker.Email()),
				UsageAttribution: customer.CustomerUsageAttribution{
					SubjectKeys: []string{RandKey()},
				},
			},
		}

		return f.Deps.CreateCustomer(inp)
	}
}

func (f CustomerFixture) GetInstanceProvider() do.Provider[InstanceProvider[customer.Customer]] {
	return func(i *do.Injector) (InstanceProvider[customer.Customer], error) {
		return func(i *do.Injector) (customer.Customer, error) {
			return f.GetProvider()(i)
		}, nil
	}
}

// Subscription

const (
	SubscriptionFixtureName         = "subscription"
	SubscriptionFixtureInstanceName = "subscription-instance"
)

type SubscriptionFixture struct {
	deps FixtureDeps
}

func (f SubscriptionFixture) Register(i *do.Injector) {
	do.ProvideNamed(i, SubscriptionFixtureName, f.GetProvider())
	do.ProvideNamed(i, SubscriptionFixtureInstanceName, f.GetInstanceProvider())
}

func (f SubscriptionFixture) GetProvider() do.Provider[subscription.Subscription] {
	return func(i *do.Injector) (subscription.Subscription, error) {
		namespace, err := do.InvokeNamed[string](i, NamespaceFixtureName)
		if err != nil {
			return subscription.Subscription{}, err
		}

		getCustomer, err := do.InvokeNamed[InstanceProvider[customer.Customer]](i, CustomerFixtureInstanceName)
		if err != nil {
			return subscription.Subscription{}, err
		}

		customer, err := getCustomer(i)
		if err != nil {
			return subscription.Subscription{}, err
		}

		return subscription.Subscription{
			NamespacedID: models.NamespacedID{
				Namespace: namespace,
				ID:        RandULID(),
			},
			CustomerId: customer.ID,
		}, nil
	}
}

func (f SubscriptionFixture) GetInstanceProvider() do.Provider[InstanceProvider[subscription.Subscription]] {
	return func(i *do.Injector) (InstanceProvider[subscription.Subscription], error) {
		return func(i *do.Injector) (subscription.Subscription, error) {
			return f.GetProvider()(i)
		}, nil
	}
}

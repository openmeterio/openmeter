package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// CustomerResolverFunc resolves customer IDs to streaming.Customer instances.
type CustomerResolverFunc func(ctx context.Context, namespace string, customerIDs []string) ([]customer.Customer, error)

// NewCustomerResolver returns a CustomerResolverFunc that uses the given customer service.
func NewCustomerResolver(customerService customer.Service) CustomerResolverFunc {
	return func(ctx context.Context, namespace string, customerIDs []string) ([]customer.Customer, error) {
		if len(customerIDs) == 0 {
			return nil, nil
		}

		customers, err := customerService.ListCustomers(ctx, customer.ListCustomersInput{
			Namespace:      namespace,
			CustomerIDs:    customerIDs,
			IncludeDeleted: false,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list customers: %w", err)
		}

		customersById := lo.KeyBy(customers.Items, func(c customer.Customer) string {
			return c.ID
		})

		var errs []error
		for _, id := range customerIDs {
			if _, ok := customersById[id]; !ok {
				errs = append(errs, NewCustomerNotFoundError(id))
			}
		}

		return customers.Items, errors.Join(errs...)
	}
}

// CustomersToStreaming converts a slice of customer.Customer to streaming.Customer.
func CustomersToStreaming(customers []customer.Customer) []streaming.Customer {
	return lo.Map(customers, func(c customer.Customer, _ int) streaming.Customer {
		return c
	})
}

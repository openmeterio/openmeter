package httpdriver

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// listCustomersBySubjectKey returns a map of customers by subject key.
func listCustomersBySubjectKey(
	ctx context.Context,
	customerService customer.Service,
	namespace string,
	subjects []string,
) (map[string]*customer.Customer, error) {
	customersBySubjectKey := map[string]*customer.Customer{}

	if len(subjects) == 0 {
		return customersBySubjectKey, nil
	}

	customers, err := listAllCustomers(ctx, customerService, customer.ListCustomersInput{
		Namespace: namespace,
		Subjects:  &subjects,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}

	for i, c := range customers {
		for _, key := range c.UsageAttribution.SubjectKeys {
			customersBySubjectKey[key] = &customers[i]
		}
	}

	return customersBySubjectKey, nil
}

// listAllCustomers returns a list of customer.
// Helper function for listing all customers. Page param will be ignored.
func listAllCustomers(ctx context.Context, service customer.Service, params customer.ListCustomersInput) ([]customer.Customer, error) {
	customers := []customer.Customer{}
	limit := 100
	page := 1

	for {
		params := params
		params.Page = pagination.NewPage(page, limit)

		result, err := service.ListCustomers(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to list all customers: %w", err)
		}

		customers = append(customers, result.Items...)

		if len(result.Items) < limit {
			break
		}

		page++
	}

	return customers, nil
}

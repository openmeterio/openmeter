package httpdriver

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListAllCustomers returns a list of customer.
// Helper function for listing all customers. Page param will be ignored.
func ListAllCustomers(ctx context.Context, service customer.Service, params customer.ListCustomersInput) ([]customer.Customer, error) {
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

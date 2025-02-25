package meter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListMetersForAllNamespaces returns a list of meters.
// Helper function for listing all meters across all namespaces.
func ListMetersForAllNamespaces(ctx context.Context, service Service) ([]Meter, error) {
	return ListAll(ctx, service, ListMetersParams{
		WithoutNamespace: true,
	})
}

// ListAll returns a list of meters.
// Helper function for listing all meters. Page param will be ignored.
func ListAll(ctx context.Context, service Service, params ListMetersParams) ([]Meter, error) {
	meters := []Meter{}
	limit := 100
	page := 1

	for {
		params := params
		params.Page = pagination.NewPage(page, limit)

		result, err := service.ListMeters(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to list all meters: %w", err)
		}

		meters = append(meters, result.Items...)

		if len(result.Items) < limit {
			break
		}

		page++
	}

	return meters, nil
}

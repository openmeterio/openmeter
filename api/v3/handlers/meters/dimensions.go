package meters

import "github.com/openmeterio/openmeter/api/v3/handlers/meters/query"

func validateDimensionsWithoutReserved[T any](dimensions map[string]T) error {
	for dimension := range dimensions {
		if query.IsReservedDimension(dimension) {
			return NewReservedDimensionError(dimension)
		}
	}

	return nil
}

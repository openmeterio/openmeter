package meters

import "github.com/openmeterio/openmeter/openmeter/meter"

const (
	dimensionSubject    = "subject"
	dimensionCustomerID = "customer_id"
)

func isReservedDimension(dimension string) bool {
	switch dimension {
	case dimensionSubject, dimensionCustomerID:
		return true
	default:
		return false
	}
}

func validateDimensionsWithoutReserved[T any](dimensions map[string]T) error {
	for dimension := range dimensions {
		if isReservedDimension(dimension) {
			return NewReservedDimensionError(dimension)
		}
	}

	return nil
}

func isSupportedGroupByDimension(m meter.Meter, dimension string) bool {
	if isReservedDimension(dimension) {
		return true
	}

	_, ok := m.GroupBy[dimension]

	return ok
}

package query

import "github.com/openmeterio/openmeter/openmeter/meter"

const (
	DimensionSubject    = "subject"
	DimensionCustomerID = "customer_id"
)

func IsReservedDimension(dimension string) bool {
	switch dimension {
	case DimensionSubject, DimensionCustomerID:
		return true
	default:
		return false
	}
}

func IsSupportedGroupByDimension(m meter.Meter, dimension string) bool {
	if IsReservedDimension(dimension) {
		return true
	}

	_, ok := m.GroupBy[dimension]

	return ok
}

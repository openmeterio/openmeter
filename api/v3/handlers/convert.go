//go:generate go tool github.com/jmattheis/goverter/cmd/goverter gen ./

package handlers

import (
	"time"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
var (
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map . CustomerMutate
	ConvertFromCreateCustomerRequestToCreateCustomerInput func(namespace string, createCustomerRequest api.CreateCustomerRequest) customer.CreateCustomerInput
	// goverter:map Metadata Labels
	// goverter:map ManagedResource.ID Id
	// goverter:map ManagedResource.Description Description
	// goverter:map ManagedResource.Name Name
	// goverter:map ManagedResource.ManagedModel.CreatedAt CreatedAt
	// goverter:map ManagedResource.ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ManagedResource.ManagedModel.DeletedAt DeletedAt
	ConvertCustomerRequestToBillingCustomer func(customer.Customer) api.BillingCustomer
	// goverter:map Labels Metadata
	// goverter:ignore Annotation
	ConvertCreateCustomerRequestToCustomerMutate func(createCustomerRequest api.CreateCustomerRequest) customer.CustomerMutate
	// goverter:map Labels Metadata
	// goverter:ignore Annotation
	// goverter:ignore Key
	ConvertUpdateCustomerRequestToCustomerMutate func(updateCustomerRequest api.UpdateCustomerRequest) customer.CustomerMutate
	ConvertCustomerListResponse                  func(customers response.CursorPaginationResponse[customer.Customer]) api.CustomerPaginatedResponse
	// goverter:map Metadata Labels
	// goverter:map GroupBy Dimensions
	// goverter:map EventType EventTypeFilter
	// goverter:map ManagedResource.ID Id
	// goverter:map ManagedResource.Description Description
	// goverter:map ManagedResource.Name Name
	// goverter:map ManagedResource.ManagedModel.CreatedAt CreatedAt
	// goverter:map ManagedResource.ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ManagedResource.ManagedModel.DeletedAt DeletedAt
	ConvertMeter func(meter.Meter) (api.Meter, error)
	// goverter:enum:unknown @error
	ConvertMeterAggregation  func(aggregation meter.MeterAggregation) (api.MeterAggregation, error)
	ConvertMeterListResponse func(meters response.CursorPaginationResponse[meter.Meter]) (api.MeterPaginatedResponse, error)
)

//goverter:context namespace
func NamespaceFromContext(namespace string) string {
	return namespace
}

type Customer struct {
	api.BillingCustomer
}

func (c Customer) Cursor() pagination.Cursor {
	return pagination.NewCursor(lo.FromPtrOr(c.CreatedAt, time.Now()), c.Id)
}

type Meter struct {
	api.Meter
}

func (m Meter) Cursor() pagination.Cursor {
	return pagination.NewCursor(lo.FromPtrOr(m.CreatedAt, time.Now()), m.Id)
}

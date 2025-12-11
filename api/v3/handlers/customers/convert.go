//go:generate go tool github.com/jmattheis/goverter/cmd/goverter gen ./

package customers

import (
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
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
	ConvertCustomerListResponse                  func(customers response.CursorPaginationResponse[customer.Customer]) api.CustomerPaginatedResponse
)

//goverter:context namespace
func NamespaceFromContext(namespace string) string {
	return namespace
}

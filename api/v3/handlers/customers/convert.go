//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package customers

import (
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend IntToFloat32
// goverter:extend ConvertMetadataAnnotationsToLabels
var (
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map . CustomerMutate
	ConvertFromCreateCustomerRequestToCreateCustomerInput func(namespace string, createCustomerRequest api.CreateCustomerRequest) customer.CreateCustomerInput
	// goverter:map . Labels | ConvertMetadataAnnotationsToLabels
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
	ConvertUpsertCustomerRequestToCustomerMutate func(updateCustomerRequest api.UpsertCustomerRequest) customer.CustomerMutate
	ConvertCustomerListResponse                  func(customers response.PagePaginationResponse[customer.Customer]) api.CustomerPagePaginatedResponse
)

//goverter:context namespace
func NamespaceFromContext(namespace string) string {
	return namespace
}

func IntToFloat32(i int) float32 {
	return float32(i)
}

func ConvertMetadataAnnotationsToLabels(source customer.Customer) *api.Labels {
	return labels.FromMetadataAnnotations(lo.FromPtr(source.Metadata), lo.FromPtr(source.Annotation))
}

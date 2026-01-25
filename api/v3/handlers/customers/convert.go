//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package customers

import (
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend IntToFloat32
// goverter:extend ConvertMetadataToLabels
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

// ConvertMetadataToLabels converts models.Metadata to api.Labels.
// Always returns an initialized map (never nil) so JSON serializes to {} instead of null.
func ConvertMetadataToLabels(source *models.Metadata) *api.Labels {
	labels := make(api.Labels)
	if source != nil {
		for k, v := range *source {
			labels[k] = v
		}
	}
	return &labels
}

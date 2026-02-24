//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package taxcodes

import (
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend ConvertMetadataToLabels
// goverter:extend IntToFloat32
var (
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map Labels Metadata
	ConvertFromCreateTaxCodeRequestToCreateTaxCodeInput func(namespace string, createTaxCodeRequest api.CreateTaxCodeRequest) (taxcode.CreateTaxCodeInput, error)

	// goverter:map Metadata Labels
	// goverter:map NamespacedID.ID Id
	// goverter:map ManagedModel.CreatedAt CreatedAt
	// goverter:map ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ManagedModel.DeletedAt DeletedAt
	ConvertTaxCodeToAPITaxCode func(taxcode.TaxCode) (api.BillingTaxCode, error)
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
func ConvertMetadataToLabels(source models.Metadata) *api.Labels {
	labels := make(api.Labels)
	for k, v := range source {
		labels[k] = v
	}
	return &labels
}

//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package taxcodes

import (
	api "github.com/openmeterio/openmeter/api/v3"
	app "github.com/openmeterio/openmeter/openmeter/app"
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
// goverter:extend ConvertAPIAppTypeToDomainAppType
// goverter:extend ConvertDomainAppTypeToAPIAppType
var (
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map Labels Metadata
	ConvertFromCreateTaxCodeRequestToCreateTaxCodeInput func(namespace string, createTaxCodeRequest api.CreateTaxCodeRequest) (taxcode.CreateTaxCodeInput, error)

	// goverter:context namespacedID
	// goverter:map NamespacedID | ResolveNamespacedIDFromContext
	// goverter:map Labels Metadata
	ConvertFromUpsertTaxCodeRequestToUpdateTaxCodeInput func(namespacedID models.NamespacedID, upsertTaxCodeRequest api.UpsertTaxCodeRequest) (taxcode.UpdateTaxCodeInput, error)

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

// goverter:context namespacedID
func ResolveNamespacedIDFromContext(namespacedID models.NamespacedID) models.NamespacedID {
	return namespacedID
}

func IntToFloat32(i int) float32 {
	return float32(i)
}

// ConvertAPIAppTypeToDomainAppType maps API app types to domain app types.
// Maps external_invoicing to custom_invoicing for backwards compatibility.
func ConvertAPIAppTypeToDomainAppType(source api.BillingAppType) app.AppType {
	if source == "external_invoicing" {
		return app.AppTypeCustomInvoicing
	}
	return app.AppType(source)
}

// ConvertDomainAppTypeToAPIAppType maps domain app types to API app types.
// Maps custom_invoicing to external_invoicing for API responses.
func ConvertDomainAppTypeToAPIAppType(source app.AppType) api.BillingAppType {
	if source == app.AppTypeCustomInvoicing {
		return "external_invoicing"
	}
	return api.BillingAppType(source)
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

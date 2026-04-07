//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package taxcodes

import (
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend ConvertAPIAppTypeToDomainAppType
// goverter:extend ConvertDomainAppTypeToAPIAppType
// goverter:extend ConvertAppMappingsToAPIAppMappings
var (
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map Labels Metadata
	// goverter:ignore Annotations
	ConvertFromCreateTaxCodeRequestToCreateTaxCodeInput func(namespace string, createTaxCodeRequest api.CreateTaxCodeRequest) (taxcode.CreateTaxCodeInput, error)

	// goverter:context namespacedID
	// goverter:map NamespacedID | ResolveNamespacedIDFromContext
	// goverter:map Labels Metadata
	// goverter:ignore Annotations
	ConvertFromUpsertTaxCodeRequestToUpdateTaxCodeInput func(namespacedID models.NamespacedID, upsertTaxCodeRequest api.UpsertTaxCodeRequest) (taxcode.UpdateTaxCodeInput, error)

	// goverter:map . Labels | ConvertTaxCodeToLabels
	// goverter:map NamespacedID.ID Id
	// goverter:map ManagedModel.CreatedAt CreatedAt
	// goverter:map ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ManagedModel.DeletedAt DeletedAt
	// goverter:map AppMappings | ConvertAppMappingsToAPIAppMappings
	ConvertTaxCodeToAPITaxCode func(taxcode.TaxCode) (api.BillingTaxCode, error)
)

var ConvertMetadataToLabels = labels.FromMetadata[models.Metadata]

//goverter:context namespace
func NamespaceFromContext(namespace string) string {
	return namespace
}

// goverter:context namespacedID
func ResolveNamespacedIDFromContext(namespacedID models.NamespacedID) models.NamespacedID {
	return namespacedID
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

// ConvertAppMappingsToAPIAppMappings converts domain app mappings to API app mappings.
// Ensures that nil is converted to an empty array instead of null.
func ConvertAppMappingsToAPIAppMappings(source taxcode.TaxCodeAppMappings) []api.BillingTaxCodeAppMapping {
	if source == nil {
		return []api.BillingTaxCodeAppMapping{}
	}

	result := make([]api.BillingTaxCodeAppMapping, len(source))
	for i, mapping := range source {
		result[i] = api.BillingTaxCodeAppMapping{
			AppType: ConvertDomainAppTypeToAPIAppType(mapping.AppType),
			TaxCode: mapping.TaxCode,
		}
	}
	return result
}

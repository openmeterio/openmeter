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
// goverter:extend FromAPIBillingAppType
// goverter:extend ToAPIBillingAppType
// goverter:extend ToAPIBillingTaxCodeAppMappings
// goverter:extend ConvertLabelsToMetadata
var (
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map Labels Metadata
	// goverter:ignore Annotations
	FromAPICreateTaxCodeRequest func(namespace string, createTaxCodeRequest api.CreateTaxCodeRequest) (taxcode.CreateTaxCodeInput, error)

	// goverter:context namespacedID
	// goverter:map NamespacedID | ResolveNamespacedIDFromContext
	// goverter:map Labels Metadata
	// goverter:ignore Annotations
	// goverter:ignore inputOptions
	FromAPIUpsertTaxCodeRequest func(namespacedID models.NamespacedID, upsertTaxCodeRequest api.UpsertTaxCodeRequest) (taxcode.UpdateTaxCodeInput, error)

	// goverter:map . Labels | ConvertMetadataAnnotationsToLabels
	// goverter:map NamespacedID.ID Id
	// goverter:map ManagedModel.CreatedAt CreatedAt
	// goverter:map ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ManagedModel.DeletedAt DeletedAt
	// goverter:map AppMappings | ToAPIBillingTaxCodeAppMappings
	ToAPIBillingTaxCode func(taxcode.TaxCode) (api.BillingTaxCode, error)
)

var ConvertLabelsToMetadata = labels.ToMetadata

func ConvertMetadataAnnotationsToLabels(source taxcode.TaxCode) *api.Labels {
	return labels.FromMetadataAnnotations(source.Metadata, source.Annotations)
}

//goverter:context namespace
func NamespaceFromContext(namespace string) string {
	return namespace
}

// goverter:context namespacedID
func ResolveNamespacedIDFromContext(namespacedID models.NamespacedID) models.NamespacedID {
	return namespacedID
}

// FromAPIBillingAppType maps API app types to domain app types.
// Maps external_invoicing to custom_invoicing for backwards compatibility.
func FromAPIBillingAppType(source api.BillingAppType) app.AppType {
	if source == "external_invoicing" {
		return app.AppTypeCustomInvoicing
	}
	return app.AppType(source)
}

// ToAPIBillingAppType maps domain app types to API app types.
// Maps custom_invoicing to external_invoicing for API responses.
func ToAPIBillingAppType(source app.AppType) api.BillingAppType {
	if source == app.AppTypeCustomInvoicing {
		return "external_invoicing"
	}
	return api.BillingAppType(source)
}

// ToAPIBillingTaxCodeAppMappings converts domain app mappings to API app mappings.
// Ensures that nil is converted to an empty array instead of null.
func ToAPIBillingTaxCodeAppMappings(source taxcode.TaxCodeAppMappings) []api.BillingTaxCodeAppMapping {
	if source == nil {
		return []api.BillingTaxCodeAppMapping{}
	}

	result := make([]api.BillingTaxCodeAppMapping, len(source))
	for i, mapping := range source {
		result[i] = api.BillingTaxCodeAppMapping{
			AppType: ToAPIBillingAppType(mapping.AppType),
			TaxCode: mapping.TaxCode,
		}
	}
	return result
}

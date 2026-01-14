//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package apps

import (
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/app"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend IntToFloat32
// goverter:enum:unknown @panic
var (
	ConvertListAppCatalogItemsResponse func(source response.PagePaginationResponse[api.BillingAppCatalogItem]) api.AppCatalogItemPagePaginatedResponse

	// goverter:autoMap Listing
	ConvertRegistryItem func(source app.RegistryItem) api.BillingAppCatalogItem

	// goverter:enum:map CapabilityTypeReportUsage BillingAppCapabilityTypeReportUsage
	// goverter:enum:map CapabilityTypeReportEvents BillingAppCapabilityTypeReportEvents
	// goverter:enum:map CapabilityTypeCalculateTax BillingAppCapabilityTypeCalculateTax
	// goverter:enum:map CapabilityTypeInvoiceCustomers BillingAppCapabilityTypeInvoiceCustomers
	// goverter:enum:map CapabilityTypeCollectPayments BillingAppCapabilityTypeCollectPayments
	ConvertCapability func(source app.CapabilityType) api.BillingAppCapabilityType

	// goverter:enum:map InstallMethodOAuth2 BillingAppInstallMethodsWithOauth2
	// goverter:enum:map InstallMethodAPIKey BillingAppInstallMethodsWithApiKey
	// goverter:enum:map InstallMethodNoCredentials BillingAppInstallMethodsNoCredentialsRequired
	ConvertInstallMethod func(source app.InstallMethod) api.BillingAppInstallMethods

	// goverter:enum:map AppTypeStripe BillingAppTypeStripe
	// goverter:enum:map AppTypeSandbox BillingAppTypeSandbox
	// goverter:enum:map AppTypeCustomInvoicing BillingAppTypeCustomInvoicing
	ConvertAppType func(source app.AppType) api.BillingAppType

	// goverter:enum:map BillingAppTypeStripe AppTypeStripe
	// goverter:enum:map BillingAppTypeSandbox AppTypeSandbox
	// goverter:enum:map BillingAppTypeCustomInvoicing AppTypeCustomInvoicing
	ConvertBillingAppType func(source api.BillingAppType) app.AppType
)

func IntToFloat32(i int) float32 {
	return float32(i)
}

//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package apps

import (
	api "github.com/openmeterio/openmeter/api"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripehttpdriver "github.com/openmeterio/openmeter/openmeter/app/stripe/httpdriver"
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
	ConvertListAppCatalogItemsResponse func(source response.PagePaginationResponse[apiv3.BillingAppCatalogItem]) apiv3.AppCatalogItemPagePaginatedResponse

	// goverter:autoMap Listing
	ConvertRegistryItem func(source app.RegistryItem) apiv3.BillingAppCatalogItem

	// goverter:enum:map CapabilityTypeReportUsage BillingAppCapabilityTypeReportUsage
	// goverter:enum:map CapabilityTypeReportEvents BillingAppCapabilityTypeReportEvents
	// goverter:enum:map CapabilityTypeCalculateTax BillingAppCapabilityTypeCalculateTax
	// goverter:enum:map CapabilityTypeInvoiceCustomers BillingAppCapabilityTypeInvoiceCustomers
	// goverter:enum:map CapabilityTypeCollectPayments BillingAppCapabilityTypeCollectPayments
	ConvertCapability func(source app.CapabilityType) apiv3.BillingAppCapabilityType

	// goverter:enum:map InstallMethodOAuth2 BillingAppInstallMethodsWithOauth2
	// goverter:enum:map InstallMethodAPIKey BillingAppInstallMethodsWithApiKey
	// goverter:enum:map InstallMethodNoCredentials BillingAppInstallMethodsNoCredentialsRequired
	ConvertInstallMethod func(source app.InstallMethod) apiv3.BillingAppInstallMethods

	// goverter:enum:map AppTypeStripe BillingAppTypeStripe
	// goverter:enum:map AppTypeSandbox BillingAppTypeSandbox
	// goverter:enum:map AppTypeCustomInvoicing BillingAppTypeCustomInvoicing
	ConvertAppType func(source app.AppType) apiv3.BillingAppType

	// goverter:enum:map BillingAppTypeStripe AppTypeStripe
	// goverter:enum:map BillingAppTypeSandbox AppTypeSandbox
	// goverter:enum:map BillingAppTypeCustomInvoicing AppTypeCustomInvoicing
	ConvertBillingAppType func(source apiv3.BillingAppType) app.AppType

	ConvertStripeWebhookResponse func(source api.StripeWebhookResponse) apiv3.BillingAppStripeWebhookResponse

	ConvertStripeWebhookRequest func(source HandleStripeWebhookRequest) appstripehttpdriver.AppStripeWebhookRequest
)

func IntToFloat32(i int) float32 {
	return float32(i)
}

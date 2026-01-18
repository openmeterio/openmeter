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
// goverter:enum:unknown @error
var (
	ConvertToListAppResponse func(source response.PagePaginationResponse[api.BillingApp]) api.AppPagePaginatedResponse

	ConvertMarketplaceListingToV3Api func(source app.MarketplaceListing) (api.BillingAppCatalogItem, error)

	// goverter:enum:map AppTypeStripe BillingAppTypeStripe
	// goverter:enum:map AppTypeSandbox BillingAppTypeSandbox
	// goverter:enum:map AppTypeCustomInvoicing BillingAppTypeExternalInvoicing
	ConvertAppTypeToV3Api func(source app.AppType) (api.BillingAppType, error)

	ConvertMetadataToLabels func(source map[string]string) *api.Labels
)

func IntToFloat32(i int) float32 {
	return float32(i)
}

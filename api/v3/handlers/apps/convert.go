//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package apps

import (
	apiv3 "github.com/openmeterio/openmeter/api/v3"
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
	ConvertToListAppResponse func(source response.PagePaginationResponse[apiv3.BillingApp]) apiv3.AppPagePaginatedResponse

	ConvertMarketplaceListingToV3Api func(source app.MarketplaceListing) apiv3.BillingAppCatalogItem

	// goverter:enum:map AppTypeStripe BillingAppTypeStripe
	// goverter:enum:map AppTypeSandbox BillingAppTypeSandbox
	// goverter:enum:map AppTypeCustomInvoicing BillingAppTypeExternalInvoicing
	ConvertAppTypeToV3Api func(source app.AppType) apiv3.BillingAppType

	ConvertMetadataToLabels func(source map[string]string) *apiv3.Labels
)

func IntToFloat32(i int) float32 {
	return float32(i)
}

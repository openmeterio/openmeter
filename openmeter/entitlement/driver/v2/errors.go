package entitlementdriverv2

import (
	"context"
	"net/http"

	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

// getErrorEncoder mirrors the v1 driver's error encoder to keep behavior consistent
func getErrorEncoder() encoder.ErrorEncoder {
	v1 := entitlementdriver.GetErrorEncoder()
	generic := commonhttp.GenericErrorEncoder()
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		if v1(ctx, err, w, r) {
			return true
		}
		return generic(ctx, err, w, r)
	}
}

package apierrors

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

// GenericErrorEncoder is an error encoder that encodes the error as a generic error.
func GenericErrorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		if err, ok := lo.ErrorsAs[*BaseAPIError](err); ok {
			err.HandleAPIError(w, r)
			return true
		}

		return commonhttp.HandleIssueIfHTTPStatusKnown(ctx, err, w)
	}
}

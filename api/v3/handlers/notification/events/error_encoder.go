package events

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

// errorEncoder maps notification-domain errors that the generic encoder cannot classify.
// Events are read-only apart from resend, so only the not-found case needs handling here;
// resend's channel-not-found and channel-disabled failures surface as generic validation
// errors and are already turned into 400s by apierrors.GenericErrorEncoder.
func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[notification.NotFoundError](ctx, http.StatusNotFound, err, w)
	}
}

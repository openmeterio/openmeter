package events

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

// errorEncoder renders notification not-found errors as v3 problem responses;
// apierrors.GenericErrorEncoder does not recognize them.
//
// Resend's channel-not-found and channel-disabled failures are generic validation
// errors and already become 400s in apierrors.GenericErrorEncoder.
func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		if _, ok := lo.ErrorsAs[notification.NotFoundError](err); !ok {
			return false
		}

		apierrors.NewNotFoundError(ctx, err, "notification event").HandleAPIError(w, r)

		return true
	}
}

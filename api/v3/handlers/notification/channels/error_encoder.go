package channels

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

// errorEncoder renders notification errors as v3 problem responses;
// apierrors.GenericErrorEncoder does not recognize them. Webhook validation
// errors expose only their public message, never the provider's error text.
func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		var apiErr *apierrors.BaseAPIError

		if validationErr, ok := lo.ErrorsAs[webhook.ValidationError](err); ok {
			apiErr = apierrors.NewBadRequestError(ctx, err, nil)
			apiErr.Detail = validationErr.PublicMessage()
		} else if _, ok := lo.ErrorsAs[notification.NotFoundError](err); ok {
			apiErr = apierrors.NewNotFoundError(ctx, err, "notification channel")
		} else if _, ok := lo.ErrorsAs[notification.UpdateAfterDeleteError](err); ok {
			apiErr = apierrors.NewConflictError(ctx, err, err.Error())
		} else {
			return false
		}

		apiErr.HandleAPIError(w, r)

		return true
	}
}

package customersentitlement

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
)

// errorEncoder renders the entitlement facade's generic domain errors as v3
// problem responses; apierrors.GenericErrorEncoder does not recognize them.
func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		switch {
		case models.IsGenericNotFoundError(err):
			apierrors.NewNotFoundError(ctx, err, "entitlement").HandleAPIError(w, r)
		case models.IsGenericPreConditionFailedError(err):
			apierrors.NewPreconditionFailedError(ctx, err.Error()).HandleAPIError(w, r)
		default:
			return false
		}

		return true
	}
}

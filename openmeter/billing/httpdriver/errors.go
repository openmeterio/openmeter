package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func errorEncoder() httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		// TODO: let's implement errors
		return commonhttp.HandleErrorIfTypeMatches[*models.GenericUserError](ctx, http.StatusBadRequest, err, w)
		/*commonhttp.HandleErrorIfTypeMatches[notification.NotFoundError](ctx, http.StatusNotFound, err, w) ||
		commonhttp.HandleErrorIfTypeMatches[*feature.FeatureNotFoundError](ctx, http.StatusBadRequest, err, w) ||
		commonhttp.HandleErrorIfTypeMatches[notification.ValidationError](ctx, http.StatusBadRequest, err, w) ||
		commonhttp.HandleErrorIfTypeMatches[webhook.ValidationError](ctx, http.StatusBadRequest, err, w) ||
		commonhttp.HandleErrorIfTypeMatches[notification.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w) ||*/
	}
}

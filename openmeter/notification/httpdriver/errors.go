package httpdriver

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
)

func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		if validationErr, ok := lo.ErrorsAs[webhook.ValidationError](err); ok {
			problem := models.NewStatusProblem(ctx, err, http.StatusBadRequest)
			problem.Detail = validationErr.PublicMessage()
			problem.Respond(w)

			return true
		}

		return commonhttp.HandleErrorIfTypeMatches[notification.NotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*feature.FeatureNotFoundError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[webhook.NotFoundError](ctx, http.StatusInternalServerError, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[notification.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w)
	}
}

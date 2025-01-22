package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/app"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func errorEncoder() httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[customerentity.NotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.ValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.SubjectKeyConflictError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.ForbiddenError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericUserError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericConflictError](ctx, http.StatusConflict, err, w) ||
			// TODO: we have to add app errors as it can happen when cutomer app data is created or updated
			commonhttp.HandleErrorIfTypeMatches[app.AppCustomerPreConditionError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppNotFoundError](ctx, http.StatusNotFound, err, w)
	}
}

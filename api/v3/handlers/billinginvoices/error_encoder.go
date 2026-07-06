package billinginvoices

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/samber/lo"
)

func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[billing.NotFoundError](ctx, http.StatusNotFound, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.ValidationError](ctx, http.StatusBadRequest, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.ValidationIssue](ctx, http.StatusBadRequest, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.AppError](ctx, http.StatusBadRequest, err, w)
	}
}

func encodeValidationIssue() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		if err == nil {
			return false
		}
		issues, convertErr := billing.ToValidationIssues(err)
		if convertErr != nil {
			return false
		}

		errs := lo.Map(issues, func(issue billing.ValidationIssue, _ int) error {
			return issue
		})

		commonhttp.NewHTTPError(http.StatusBadRequest, errors.Join(errs...)).EncodeError(ctx, w)
		return true
	}
}

package billinginvoices

import (
	"context"
	"errors"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/handlers/billingerrors"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

func errorEncoder() encoder.ErrorEncoder {
	return billingerrors.ErrorEncoder()
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

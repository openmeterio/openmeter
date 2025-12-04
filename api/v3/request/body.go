package request

import (
	"encoding/json"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

func ParseBody(r *http.Request, payload any) *apierrors.BaseAPIError {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		return apierrors.NewBadRequestError(r.Context(), err,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Reason: "unable to parse body",
					Source: apierrors.InvalidParamSourceBody,
				},
			},
		)
	}

	return nil
}

package request

import (
	"encoding/json"
	"errors"
	"io"
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

// ParseOptionalBody parses the request body if present, leaving payload unchanged if the body is empty.
func ParseOptionalBody(r *http.Request, payload any) *apierrors.BaseAPIError {
	if r.Body == nil {
		return nil
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

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

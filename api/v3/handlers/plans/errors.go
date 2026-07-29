package plans

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

// asBadRequestIfConversionError maps request-conversion failures that are
// caused by malformed input to a 400, leaving anything else untouched.
//
// The converters return plain errors, which the v3 error encoder does not
// recognise: it handles *apierrors.BaseAPIError, a couple of not-found types,
// and validation issues carrying an HTTP status attribute. Anything else falls
// through and surfaces as a bare 500 with no body, which is what a usage-based
// rate card missing billing_cadence used to produce.
func asBadRequestIfConversionError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	var cadenceErr ErrRateCardBillingCadenceRequired
	if errors.As(err, &cadenceErr) {
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "phases[].rate_cards[].billing_cadence",
				Rule:   "required",
				Reason: cadenceErr.Error(),
				Source: apierrors.InvalidParamSourceBody,
			},
		})
	}

	return err
}

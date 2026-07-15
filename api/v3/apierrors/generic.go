package apierrors

import (
	"context"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
)

// mapGenericError converts model-level errors into the rich v3 error shape so
// every recognized domain error carries a distinct type URI instead of falling
// back to the legacy "about:blank" problem shape. It returns nil when it has
// no mapping, leaving the caller to fall through to the legacy encoder.
//
// Detail is taken from the matched error's own message: with the exception of
// authorization errors (which keep their fixed, non-revealing details), domain
// call sites wrap curated messages that are safe and useful to surface.
func mapGenericError(ctx context.Context, err error) *BaseAPIError {
	// Validation issues carrying an explicit HTTP status are the most specific
	// classification available, so they take precedence over the generic
	// wrapper types below.
	if status, ok := singularHTTPStatusFromValidationIssues(err); ok {
		if mapped := apiErrorFromHTTPStatus(ctx, status, err); mapped != nil {
			return mapped
		}
	}

	if conflict, ok := lo.ErrorsAs[*models.GenericConflictError](err); ok {
		var opts []ConflictOption
		if resource := conflict.Resource(); resource != nil {
			opts = append(opts, WithConflictingResource(ConflictingResource{
				Type:       resource.Type,
				ID:         resource.ID,
				CustomerID: resource.CustomerID,
			}))
		}
		return NewConflictError(ctx, err, conflict.Error(), opts...)
	}

	if notFound, ok := lo.ErrorsAs[*models.GenericNotFoundError](err); ok {
		mapped := NewNotFoundError(ctx, err, "")
		mapped.Detail = notFound.Error()
		return mapped
	}

	if validation, ok := lo.ErrorsAs[*models.GenericValidationError](err); ok {
		mapped := NewBadRequestError(ctx, err, nil)
		mapped.Detail = validation.Error()
		return mapped
	}

	if precondition, ok := lo.ErrorsAs[*models.GenericPreConditionFailedError](err); ok {
		return NewPreconditionFailedError(ctx, precondition.Error())
	}

	if _, ok := lo.ErrorsAs[*models.GenericForbiddenError](err); ok {
		return NewForbiddenError(ctx, err)
	}

	if _, ok := lo.ErrorsAs[*models.GenericUnauthorizedError](err); ok {
		return NewUnauthenticatedError(ctx, err)
	}

	if _, ok := lo.ErrorsAs[*models.GenericNotImplementedError](err); ok {
		return NewNotImplementedError(ctx, err)
	}

	if featureNotFound, ok := lo.ErrorsAs[*feature.FeatureNotFoundError](err); ok {
		mapped := NewNotFoundError(ctx, err, "feature")
		mapped.Detail = featureNotFound.Error()
		return mapped
	}

	if meterNotFound, ok := lo.ErrorsAs[*meter.MeterNotFoundError](err); ok {
		mapped := NewNotFoundError(ctx, err, "meter")
		mapped.Detail = meterNotFound.Error()
		return mapped
	}

	return nil
}

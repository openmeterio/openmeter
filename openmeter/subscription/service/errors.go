package service

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

// mapSubscriptionErrors maps subscription errors to user errors
func mapSubscriptionErrors(err error) error {
	if err == nil {
		return nil
	}

	if sErr, ok := lo.ErrorsAs[*subscription.SpecValidationError](err); ok {
		return models.NewGenericValidationError(sErr)
	} else if sErr, ok := lo.ErrorsAs[*subscription.AlignmentError](err); ok {
		return models.NewGenericConflictError(sErr)
	} else if sErr, ok := lo.ErrorsAs[*subscription.NoBillingPeriodError](err); ok {
		return models.NewGenericValidationError(sErr)
	}

	return err
}

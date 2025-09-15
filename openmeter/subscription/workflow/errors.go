package subscriptionworkflow

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

func MapSubscriptionErrors(err error) error {
	if err == nil {
		return nil
	}

	if sErr, ok := lo.ErrorsAs[*subscription.SpecValidationError](err); ok {
		return models.NewGenericValidationError(sErr)
	} else if sErr, ok := lo.ErrorsAs[*subscription.AlignmentError](err); ok {
		return models.NewGenericConflictError(sErr)
	}

	return err
}

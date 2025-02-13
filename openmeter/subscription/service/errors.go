package service

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

// mapSubscriptionErrors maps subscription errors to user errors
func mapSubscriptionErrors(err error) (error, bool) {
	if sErr, ok := lo.ErrorsAs[*subscription.SpecValidationError](err); ok {
		return &models.GenericUserError{Inner: sErr}, true
	} else if sErr, ok := lo.ErrorsAs[*subscription.AlignmentError](err); ok {
		return &models.GenericUserError{Inner: sErr}, true
	}

	return err, false
}

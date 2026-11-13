package subscriptionsync

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// ListDeletedSubscriptionsToReconcileInput selects deleted subscriptions whose
// gathering lines still need cleanup. Only subscriptions that have not started
// can be deleted, but sync may already have generated their future gathering lines.
// Billing collection can pick up those lines while deletion cleanup is pending,
// so cleanup must not wait for ActiveFrom or DeletedAt to pass. A successful v2
// subscription_deleted result is what stops further cleanup attempts.
type ListDeletedSubscriptionsToReconcileInput struct {
	Namespaces []string
	Customers  []string
	// Cursor pages by creation time and subscription ID. Offset pagination could
	// skip subscriptions when concurrent cleanup removes earlier candidates.
	Cursor *paginationv2.Cursor
	Limit  int
}

func (i ListDeletedSubscriptionsToReconcileInput) Validate() error {
	var errs []error
	if i.Limit <= 0 {
		errs = append(errs, errors.New("limit must be greater than zero"))
	}

	if i.Cursor != nil {
		if err := i.Cursor.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("cursor: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

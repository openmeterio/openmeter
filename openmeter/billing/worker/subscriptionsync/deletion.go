package subscriptionsync

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// ListDeletedSubscriptionsToReconcileInput selects subscriptions deleted by AsOf
// without a successful subscription_deleted result. Service dates and legacy sync
// state do not establish that deletion cleanup has completed.
type ListDeletedSubscriptionsToReconcileInput struct {
	Namespaces []string
	Customers  []string
	AsOf       time.Time
	// Cursor pages by creation time and subscription ID. Offset pagination could
	// skip subscriptions when concurrent cleanup removes earlier candidates.
	Cursor *paginationv2.Cursor
	Limit  int
}

func (i ListDeletedSubscriptionsToReconcileInput) Validate() error {
	var errs []error
	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

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

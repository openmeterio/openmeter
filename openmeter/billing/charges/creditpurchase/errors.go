package creditpurchase

import (
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeCreditPurchaseChargeNotActive models.ErrorCode = "credit_purchase_charge_not_active"

var ErrCreditPurchaseChargeNotActive = models.NewValidationIssue(
	ErrCodeCreditPurchaseChargeNotActive,
	"credit purchase charge is not active",
	models.WithFieldString("namespace"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// ChargeKeyConflictError is returned when a create request's idempotency key
// collides with a live (non-deleted) credit purchase charge. It carries the
// coordinates needed to look up the conflicting charge later; the charge's own
// ID is deliberately not resolved here, because the failed insert leaves the
// surrounding transaction in an aborted state until it unwinds, so no further
// reads are possible at the detection site.
type ChargeKeyConflictError struct {
	err        error
	Namespace  string
	CustomerID string
	Key        string
}

func NewChargeKeyConflictError(namespace, customerID, key string) error {
	return &ChargeKeyConflictError{
		err: models.NewGenericConflictError(
			fmt.Errorf("credit purchase charge with key %q already exists for customer %s", key, customerID),
		),
		Namespace:  namespace,
		CustomerID: customerID,
		Key:        key,
	}
}

func (e *ChargeKeyConflictError) Error() string {
	return e.err.Error()
}

func (e *ChargeKeyConflictError) Unwrap() error {
	return e.err
}

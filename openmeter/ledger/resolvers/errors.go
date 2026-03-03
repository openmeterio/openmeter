package resolvers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

type validationErrors interface {
	ValidationErrors() (models.ValidationIssues, error)
}

const ErrCodeCustomerAccountConflict models.ErrorCode = "customer_account_conflict"

var ErrCustomerAccountConflict = models.NewValidationIssue(
	ErrCodeCustomerAccountConflict,
	"customer account mapping conflict, a mapping with the same customer and account type already exists",
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

type CustomerAccountAlreadyExistsError struct {
	CustomerID  customer.CustomerID
	AccountType ledger.AccountType
	AccountID   string
}

var _ validationErrors = (*CustomerAccountAlreadyExistsError)(nil)

func (e *CustomerAccountAlreadyExistsError) Error() string {
	return fmt.Sprintf(
		"customer account mapping already exists: namespace=%s customer_id=%s account_type=%s account_id=%s",
		e.CustomerID.Namespace,
		e.CustomerID.ID,
		e.AccountType,
		e.AccountID,
	)
}

func (e *CustomerAccountAlreadyExistsError) ValidationErrors() (models.ValidationIssues, error) {
	return models.ValidationIssues{
		ErrCustomerAccountConflict.WithAttrs(models.Attributes{
			"namespace":    e.CustomerID.Namespace,
			"customer_id":  e.CustomerID.ID,
			"account_type": e.AccountType,
			"account_id":   e.AccountID,
		}),
	}, nil
}

func AsCustomerAccountAlreadyExistsError(err error) (*CustomerAccountAlreadyExistsError, bool) {
	var target *CustomerAccountAlreadyExistsError
	if errors.As(err, &target) {
		return target, true
	}

	return nil, false
}

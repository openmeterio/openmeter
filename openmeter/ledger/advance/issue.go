package advance

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/crediteligibility"
	"github.com/openmeterio/openmeter/pkg/models"
)

type IssueInput struct {
	CustomerID  customer.CustomerID
	ChargeID    string
	At          time.Time
	Amount      alpacadecimal.Decimal
	Currency    currencies.CurrencyReference
	Filters     crediteligibility.Filters
	TaxCode     *string
	TaxBehavior *ledger.TaxBehavior
}

func (i IssueInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}

	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("amount: %w", err))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	} else if i.Currency.IsCustom() && !i.Currency.IsResolved() {
		errs = append(errs, errors.New("custom currency must be resolved"))
	}

	if err := i.Filters.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("filters: %w", err))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

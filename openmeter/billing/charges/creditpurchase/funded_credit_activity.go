package creditpurchase

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type FundedCreditActivityCursor struct {
	FundedAt        time.Time
	ChargeCreatedAt time.Time
	ChargeID        meta.ChargeID
}

func (c FundedCreditActivityCursor) Validate() error {
	var errs []error

	if c.FundedAt.IsZero() {
		errs = append(errs, fmt.Errorf("funded_at is required"))
	}

	if c.ChargeCreatedAt.IsZero() {
		errs = append(errs, fmt.Errorf("charge_created_at is required"))
	}

	if err := c.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge_id: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type FundedCreditActivity struct {
	ChargeID           meta.ChargeID
	ChargeCreatedAt    time.Time
	FundedAt           time.Time
	TransactionGroupID string
	Currency           currencyx.Code
	Amount             alpacadecimal.Decimal
	Name               string
	Description        *string
}

type ListFundedCreditActivitiesInput struct {
	Customer customer.CustomerID
	Limit    int
	After    *FundedCreditActivityCursor
	Before   *FundedCreditActivityCursor
	Currency *currencyx.Code
}

func (i ListFundedCreditActivitiesInput) Validate() error {
	var errs []error

	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if i.Limit < 1 {
		errs = append(errs, fmt.Errorf("limit must be greater than 0"))
	}

	if i.After != nil {
		if err := i.After.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("after: %w", err))
		}
	}

	if i.Before != nil {
		if err := i.Before.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("before: %w", err))
		}
	}

	if i.After != nil && i.Before != nil {
		errs = append(errs, fmt.Errorf("after and before cannot be set together"))
	}

	if i.Currency != nil {
		if err := i.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ListFundedCreditActivitiesResult struct {
	Items       []FundedCreditActivity
	NextCursor  *FundedCreditActivityCursor
	HasPrevious bool
}

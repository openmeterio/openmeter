package rte

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreditBalanceKey identifies a current balance, not a historical query.
type CreditBalanceKey struct {
	CustomerID    customer.CustomerID
	Currency      currencies.CurrencyReference
	FeatureFilter mo.Option[creditpurchase.FeatureFilters]
}

func (k CreditBalanceKey) Validate() error {
	var errs []error

	if err := k.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := k.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if k.Currency.IsCustom() && (k.Currency.CustomCurrencyID == nil || *k.Currency.CustomCurrencyID == "") {
		errs = append(errs, errors.New("custom currency ID is required"))
	}

	if err := customerbalance.ValidateFeatureFilter(k.FeatureFilter); err != nil {
		errs = append(errs, fmt.Errorf("feature filter: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// FeatureScope preserves the difference between the whole portfolio,
// unrestricted credits, and credits eligible for one feature.
func (k CreditBalanceKey) FeatureScope() string {
	if k.FeatureFilter.IsAbsent() {
		return "all"
	}

	features := k.FeatureFilter.OrEmpty().Normalize()
	if len(features) == 0 {
		return "unrestricted"
	}

	return "feature:" + features[0]
}

type CreditBalance struct {
	// CalculatedAt is captured before calculation. Only a strictly later start
	// replaces stored state; ordering across replicas relies on their clocks.
	CalculatedAt time.Time             `json:"calculated_at"`
	Booked       alpacadecimal.Decimal `json:"booked"`
	Spendable    alpacadecimal.Decimal `json:"spendable"`
	Live         alpacadecimal.Decimal `json:"live"`
	Pending      alpacadecimal.Decimal `json:"pending"`
}

type StoreCreditBalanceInput struct {
	Key     CreditBalanceKey
	Balance CreditBalance
}

func (i StoreCreditBalanceInput) Validate() error {
	var errs []error

	if err := i.Key.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Balance.CalculatedAt.IsZero() {
		errs = append(errs, errors.New("calculated at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Service interface {
	CalculateCreditBalance(context.Context, CreditBalanceKey) (CreditBalance, error)
	GetCreditBalance(context.Context, CreditBalanceKey) (CreditBalance, error)
}

type Adapter interface {
	// StoreCreditBalance replaces state only for a strictly later CalculatedAt.
	// Equal or older results leave the existing success intact and return it.
	StoreCreditBalance(context.Context, StoreCreditBalanceInput) (CreditBalance, error)
	GetCreditBalance(context.Context, CreditBalanceKey) (CreditBalance, error)
}

func NewCreditBalanceNotCalculatedError() error {
	return models.NewGenericNotFoundError(errors.New("credit balance has not been calculated"))
}

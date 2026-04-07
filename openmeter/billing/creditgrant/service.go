package creditgrant

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Service provides a credit-grant-centric API on top of the charges layer.
type Service interface {
	Create(ctx context.Context, input CreateInput) (creditpurchase.Charge, error)
	Get(ctx context.Context, input GetInput) (creditpurchase.Charge, error)
	List(ctx context.Context, input ListInput) (pagination.Result[creditpurchase.Charge], error)
}

// FundingMethod represents how a credit grant is funded.
type FundingMethod string

const (
	FundingMethodNone     FundingMethod = "none"
	FundingMethodInvoice  FundingMethod = "invoice"
	FundingMethodExternal FundingMethod = "external"
)

func (f FundingMethod) Validate() error {
	switch f {
	case FundingMethodNone, FundingMethodInvoice, FundingMethodExternal:
		return nil
	default:
		return fmt.Errorf("invalid funding method: %s", f)
	}
}

// PurchaseTerms defines the purchase/payment terms for a credit grant.
type PurchaseTerms struct {
	Currency           currencyx.Code
	PerUnitCostBasis   *alpacadecimal.Decimal
	AvailabilityPolicy *creditpurchase.InitialPaymentSettlementStatus
}

type CreateInput struct {
	Namespace   string
	CustomerID  string
	Name        string
	Description *string
	Labels      map[string]string
	// TODO: support custom currency codes later
	Currency      currencyx.Code
	Amount        alpacadecimal.Decimal
	Priority      *int16
	FundingMethod FundingMethod
	Purchase      *PurchaseTerms
	TaxConfig     *productcatalog.TaxConfig
	Filters       *GrantFilters
}

type GrantFilters struct {
	Features []string
}

func (i CreateInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer ID is required"))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if !i.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}

	if err := i.FundingMethod.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.FundingMethod != FundingMethodNone && i.Purchase == nil {
		errs = append(errs, errors.New("purchase terms are required for funded grants"))
	}

	if i.Purchase != nil {
		if err := i.Purchase.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("purchase currency: %w", err))
		}

		if i.Purchase.PerUnitCostBasis != nil && !i.Purchase.PerUnitCostBasis.IsPositive() {
			errs = append(errs, errors.New("per_unit_cost_basis must be positive"))
		}
	}

	if i.TaxConfig != nil {
		if err := i.TaxConfig.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax config: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetInput struct {
	Namespace  string
	CustomerID string
	ChargeID   string
}

func (i GetInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer ID is required"))
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ListInput struct {
	pagination.Page

	Namespace  string
	CustomerID string

	// Optional filters
	Status   *meta.ChargeStatus
	Currency *currencyx.Code
}

func (i ListInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer ID is required"))
	}

	if i.Status != nil {
		if err := i.Status.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Currency != nil {
		if err := i.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

package feature

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
)

// CostKind specifies the cost type
type CostKind string

const (
	// CostKindManual manual cost input
	CostKindManual CostKind = "manual"
	// CostKindProvider
	CostKindProvider CostKind = "provider"
)

func (k CostKind) Values() []string {
	return []string{
		string(CostKindManual),
		string(CostKindProvider),
	}
}

// Cost holds data for cost calculation
type Cost struct {
	// The cost type
	Kind CostKind `json:"kind"`

	// Provider ID, only if type "provider"
	ProviderID *string `json:"providerId,omitempty"`

	// Currency for cost
	Currency currency.Code `json:"currency"`

	// The per unit amount, Wwriteable only if type "manual"
	PerUnitAmount alpacadecimal.Decimal `json:"costPerUnit,omitempty"`
}

// Validate validates the cost.
func (a *Cost) Validate() error {
	var errs []error

	if a.Kind == "" {
		errs = append(errs, fmt.Errorf("kind is required"))
	}

	if a.Kind == CostKindManual {
		if a.Currency == "" {
			errs = append(errs, fmt.Errorf("currency is required with kind manual"))
		}

		if a.PerUnitAmount.IsZero() {
			errs = append(errs, fmt.Errorf("per unit amount is required with kind manual"))
		}
	}

	if a.Kind == CostKindProvider {
		if *a.ProviderID == "" {
			errs = append(errs, fmt.Errorf("provider id is required with kind provider"))
		}
	}

	return errors.Join(errs...)
}

// Cost holds data for cost calculation
type CostMutateInput struct {
	// The cost type
	Kind CostKind `json:"kind"`

	// Provider ID, only if type "provider"
	ProviderID *string `json:"providerId,omitempty"`

	// Currency for cost
	Currency currency.Code `json:"currency"`

	// The per unit amount, Wwriteable only if type "manual"
	PerUnitAmount *alpacadecimal.Decimal `json:"costPerUnit,omitempty"`
}

// Validate validates the cost.
func (a *CostMutateInput) Validate() error {
	var errs []error

	if a.Kind == "" {
		errs = append(errs, fmt.Errorf("kind is required"))
	}

	if a.Kind == CostKindManual {
		if a.Currency == "" {
			errs = append(errs, fmt.Errorf("currency is required with kind manual"))
		}

		if a.PerUnitAmount.IsZero() {
			errs = append(errs, fmt.Errorf("per unit amount is required with kind manual"))
		}
	}

	if a.Kind == CostKindProvider {
		if *a.ProviderID == "" {
			errs = append(errs, fmt.Errorf("provider id is required with kind provider"))
		}
	}

	return errors.Join(errs...)
}

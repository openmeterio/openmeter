package creditreconciliation

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeCreditAllocationsDoNotMatchTotal models.ErrorCode = "credit_allocations_do_not_match_total"

var ErrCreditAllocationsDoNotMatchTotal = models.NewValidationIssue(
	ErrCodeCreditAllocationsDoNotMatchTotal,
	"credit allocations do not match total",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// AllocationHandler binds allocation to one charge type and monetary domain.
// It owns domain validation, ledger effects, persistence, and realization
// lineage without requiring an existing realization history.
type AllocationHandler interface {
	Validate() error
	CurrencyCalculator() currencyx.Currency
	Allocate(context.Context, alpacadecimal.Decimal) (creditrealization.CreateAllocationInputs, error)
	Create(context.Context, creditrealization.CreateInputs) (creditrealization.Realizations, error)
}

// Handler extends allocation with the realization history and correction
// behavior required by reconciliation.
type Handler interface {
	AllocationHandler
	Realizations() creditrealization.Realizations
	Correct(
		context.Context,
		creditrealization.CorrectionRequest,
	) (creditrealization.CreateCorrectionInputs, error)
}

type AllocateInput struct {
	Amount          alpacadecimal.Decimal
	ExactAllocation bool
	Handler         AllocationHandler
}

func (i AllocateInput) Validate() error {
	var errs []error

	if i.Amount.IsNegative() {
		errs = append(errs, errors.New("amount must be zero or positive"))
	}

	if i.Handler == nil {
		errs = append(errs, errors.New("credit allocation handler is required"))
	} else {
		if err := i.Handler.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit allocation handler: %w", err))
		}

		currencyCalculator := i.Handler.CurrencyCalculator()
		if currencyCalculator == nil {
			errs = append(errs, errors.New("credit allocation handler currency calculator is required"))
		} else if err := currencyCalculator.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit allocation handler currency calculator: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type AllocateResult struct {
	AllocatedAmount alpacadecimal.Decimal
	Realizations    creditrealization.Realizations
}

// Allocate creates up to Amount in new credit allocations. It never reads or
// corrects existing realizations, making it suitable for one-shot lifecycle
// effects whose caller owns the empty-history invariant.
func Allocate(ctx context.Context, in AllocateInput) (AllocateResult, error) {
	if err := in.Validate(); err != nil {
		return AllocateResult{}, err
	}

	currencyCalculator := in.Handler.CurrencyCalculator()
	in.Amount = currencyCalculator.RoundToPrecision(in.Amount)
	if in.Amount.IsZero() {
		return AllocateResult{}, nil
	}

	allocations, err := in.Handler.Allocate(ctx, in.Amount)
	if err != nil {
		return AllocateResult{}, err
	}

	allocated := currencyCalculator.RoundToPrecision(allocations.Sum())
	if allocated.GreaterThan(in.Amount) || (in.ExactAllocation && !allocated.Equal(in.Amount)) {
		return AllocateResult{}, ErrCreditAllocationsDoNotMatchTotal.WithAttrs(models.Attributes{
			"total": in.Amount.String(),
		})
	}

	result := AllocateResult{AllocatedAmount: allocated}
	if len(allocations) == 0 {
		return result, nil
	}

	realizations, err := in.Handler.Create(ctx, allocations.AsCreateInputs())
	if err != nil {
		return AllocateResult{}, err
	}

	result.Realizations = realizations

	return result, nil
}

type ReconcileInput struct {
	TargetAmount    alpacadecimal.Decimal
	ExactAllocation bool
	Handler         Handler
}

func (i ReconcileInput) Validate() error {
	var errs []error

	if i.TargetAmount.IsNegative() {
		errs = append(errs, errors.New("target amount must be zero or positive"))
	}

	if i.Handler == nil {
		errs = append(errs, errors.New("credit reconciliation handler is required"))
	} else {
		if err := i.Handler.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit reconciliation handler: %w", err))
		}

		currencyCalculator := i.Handler.CurrencyCalculator()
		if currencyCalculator == nil {
			errs = append(errs, errors.New("credit reconciliation handler currency calculator is required"))
		} else if err := currencyCalculator.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit reconciliation handler currency calculator: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ReconcileResult struct {
	Delta        alpacadecimal.Decimal
	Realizations creditrealization.Realizations
}

// Reconcile adjusts the handler's credit realizations to TargetAmount.
// Positive deltas allocate credits, negative deltas create corrections, and a
// zero delta deliberately produces no new realization.
func Reconcile(ctx context.Context, in ReconcileInput) (ReconcileResult, error) {
	if err := in.Validate(); err != nil {
		return ReconcileResult{}, err
	}

	currencyCalculator := in.Handler.CurrencyCalculator()
	in.TargetAmount = currencyCalculator.RoundToPrecision(in.TargetAmount)
	if err := in.Validate(); err != nil {
		return ReconcileResult{}, err
	}

	currentRealizations := in.Handler.Realizations()
	currentAmount := currencyCalculator.RoundToPrecision(currentRealizations.Sum())
	delta := currencyCalculator.RoundToPrecision(in.TargetAmount.Sub(currentAmount))
	result := ReconcileResult{Delta: delta}

	switch {
	case delta.IsPositive():
		allocationResult, err := Allocate(ctx, AllocateInput{
			Amount:          delta,
			ExactAllocation: in.ExactAllocation,
			Handler:         in.Handler,
		})
		if err != nil {
			return ReconcileResult{}, err
		}

		result.Realizations = allocationResult.Realizations
	case delta.IsNegative():
		corrections, err := currentRealizations.Correct(
			delta,
			currencyCalculator,
			func(request creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
				return in.Handler.Correct(ctx, request)
			},
		)
		if err != nil {
			return ReconcileResult{}, err
		}

		if len(corrections) > 0 {
			realizations, err := in.Handler.Create(ctx, corrections)
			if err != nil {
				return ReconcileResult{}, err
			}

			result.Realizations = realizations
		}
	case delta.IsZero():
	}

	return result, nil
}

type CorrectAllInput struct {
	Handler Handler
}

// CorrectAll reverses every active allocation selected by Handler.
func CorrectAll(ctx context.Context, in CorrectAllInput) (ReconcileResult, error) {
	return Reconcile(ctx, ReconcileInput{
		TargetAmount: alpacadecimal.Zero,
		Handler:      in.Handler,
	})
}

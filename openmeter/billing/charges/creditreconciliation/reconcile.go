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

// Handler binds reconciliation to one charge type and monetary domain. It owns
// domain validation, ledger effects, persistence, and realization lineage.
type Handler interface {
	Validate() error
	CurrencyCalculator() currencyx.Currency
	Realizations() creditrealization.Realizations
	Allocate(context.Context, alpacadecimal.Decimal) (creditrealization.CreateAllocationInputs, error)
	Correct(
		context.Context,
		creditrealization.CorrectionRequest,
	) (creditrealization.CreateCorrectionInputs, error)
	Create(context.Context, creditrealization.CreateInputs) (creditrealization.Realizations, error)
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
		allocations, err := in.Handler.Allocate(ctx, delta)
		if err != nil {
			return ReconcileResult{}, err
		}

		allocated := currencyCalculator.RoundToPrecision(allocations.Sum())
		if allocated.GreaterThan(delta) || (in.ExactAllocation && !allocated.Equal(delta)) {
			return ReconcileResult{}, ErrCreditAllocationsDoNotMatchTotal.WithAttrs(models.Attributes{
				"total": delta.String(),
			})
		}

		if len(allocations) > 0 {
			realizations, err := in.Handler.Create(ctx, allocations.AsCreateInputs())
			if err != nil {
				return ReconcileResult{}, err
			}

			result.Realizations = realizations
		}
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

package realizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type StartCreditThenInvoiceRunInput struct {
	Charge    flatfee.Charge
	LineID    string
	InvoiceID string
}

func (i StartCreditThenInvoiceRunInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.LineID == "" {
		errs = append(errs, errors.New("line ID is required"))
	}

	if i.InvoiceID == "" {
		errs = append(errs, errors.New("invoice ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *Service) StartCreditThenInvoiceRun(ctx context.Context, in StartCreditThenInvoiceRunInput) (flatfee.RealizationRun, error) {
	if err := in.Validate(); err != nil {
		return flatfee.RealizationRun{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (flatfee.RealizationRun, error) {
		rateableIntent, err := in.Charge.GetRateableIntent()
		if err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("getting rateable intent: %w", err)
		}

		ratingResult, err := s.Rate(rateableIntent)
		if err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("rating flat fee: %w", err)
		}

		runBase, err := s.adapter.CreateCurrentRun(ctx, flatfee.CreateCurrentRunInput{
			Charge:                    in.Charge.ChargeBase,
			ServicePeriod:             rateableIntent.ServicePeriod,
			AmountAfterProration:      rateableIntent.AmountAfterProration,
			NoFiatTransactionRequired: ratingResult.Totals.Total.IsZero(),
			Immutable:                 false,
			LineID:                    lo.ToPtr(in.LineID),
			InvoiceID:                 lo.ToPtr(in.InvoiceID),
		})
		if err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("create current run: %w", err)
		}

		charge := in.Charge
		run := flatfee.RealizationRun{
			RealizationRunBase: runBase,
		}
		charge.Realizations.CurrentRun = &run

		reconciledRun, err := s.ReconcileRatedRun(ctx, ReconcileRatedRunInput{
			Charge:             charge,
			Run:                run,
			Rating:             ratingResult,
			CurrencyCalculator: charge.Intent.GetCurrency(),
		})
		if err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("reconcile rated run: %w", err)
		}

		return reconciledRun, nil
	})
}

type ReconcileRunToIntentInput struct {
	Charge flatfee.Charge
	Run    flatfee.RealizationRun
}

func (i ReconcileRunToIntentInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.Run.LineID == nil || *i.Run.LineID == "" {
		errs = append(errs, errors.New("run line ID is required"))
	}

	if i.Run.InvoiceID == nil || *i.Run.InvoiceID == "" {
		errs = append(errs, errors.New("run invoice ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ReconcileRunToIntent rerates a mutable credit_then_invoice run from
// the effective charge intent and reconciles its credit allocations.
func (s *Service) ReconcileRunToIntent(ctx context.Context, in ReconcileRunToIntentInput) (flatfee.RealizationRun, error) {
	if err := in.Validate(); err != nil {
		return flatfee.RealizationRun{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (flatfee.RealizationRun, error) {
		rateableIntent, err := in.Charge.GetRateableIntent()
		if err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("getting rateable intent: %w", err)
		}

		ratingResult, err := s.Rate(rateableIntent)
		if err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("rating flat fee: %w", err)
		}

		reconciledRun, err := s.ReconcileRatedRun(ctx, ReconcileRatedRunInput{
			Charge:             in.Charge,
			Run:                in.Run,
			Rating:             ratingResult,
			CurrencyCalculator: in.Charge.Intent.GetCurrency(),
		})
		if err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("reconcile rated run: %w", err)
		}

		return reconciledRun, nil
	})
}

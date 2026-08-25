package realizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AccrueInvoiceUsageInput struct {
	Charge         flatfee.Charge
	LineWithHeader billing.StandardLineWithInvoiceHeader
}

func (i AccrueInvoiceUsageInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.LineWithHeader.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("line with header: %w", err))
	}

	if i.Charge.Realizations.CurrentRun == nil {
		errs = append(errs, fmt.Errorf("current run is required"))
	} else {
		currentRun := i.Charge.Realizations.CurrentRun

		if currentRun.AccruedUsage != nil {
			errs = append(errs, fmt.Errorf("accrued invoice usage already exists for charge %s", i.Charge.GetChargeID()))
		}

		if i.LineWithHeader.Line != nil {
			if currentRun.LineID == nil || *currentRun.LineID != i.LineWithHeader.Line.ID {
				errs = append(errs, fmt.Errorf("current run line id must match standard line"))
			}

			if currentRun.NoFiatTransactionRequired && !i.LineWithHeader.Line.Totals.Total.IsZero() {
				errs = append(errs, fmt.Errorf("current run requires no fiat transaction but line total is non-zero"))
			}

			if !currentRun.NoFiatTransactionRequired && i.LineWithHeader.Line.Totals.Total.IsZero() {
				errs = append(errs, fmt.Errorf("current run has zero line total but requires a fiat transaction"))
			}
		}

		if currentRun.InvoiceID == nil || *currentRun.InvoiceID != i.LineWithHeader.Invoice.ID {
			errs = append(errs, fmt.Errorf("current run invoice id must match invoice"))
		}
	}

	if i.LineWithHeader.Line != nil {
		if i.LineWithHeader.Line.ChargeID == nil || *i.LineWithHeader.Line.ChargeID != i.Charge.ID {
			errs = append(errs, fmt.Errorf("line charge id must match charge"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type AccrueInvoiceUsageResult struct {
	AccruedUsage *invoicedusage.AccruedUsage
	Run          flatfee.RealizationRun
}

func (s *Service) AccrueInvoiceUsage(ctx context.Context, in AccrueInvoiceUsageInput) (AccrueInvoiceUsageResult, error) {
	if err := in.Validate(); err != nil {
		return AccrueInvoiceUsageResult{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (AccrueInvoiceUsageResult, error) {
		currentRun := *in.Charge.Realizations.CurrentRun
		line := *in.LineWithHeader.Line

		result := AccrueInvoiceUsageResult{
			Run: currentRun,
		}

		if !currentRun.NoFiatTransactionRequired {
			var ledgerTransactionRef ledgertransaction.GroupReference
			if in.Charge.Intent.GetCurrency().IsCustom() {
				handlerInput := flatfee.OnCustomCurrencyOverageAccruedInput{
					Charge: in.Charge,
					Run:    currentRun,
				}
				if err := handlerInput.Validate(); err != nil {
					return AccrueInvoiceUsageResult{}, fmt.Errorf("validating custom currency overage accrued input: %w", err)
				}

				handlerResult, err := s.handler.OnCustomCurrencyOverageAccrued(ctx, handlerInput)
				if err != nil {
					return AccrueInvoiceUsageResult{}, fmt.Errorf("on flat fee custom currency overage accrued: %w", err)
				}
				if err := handlerResult.Validate(); err != nil {
					return AccrueInvoiceUsageResult{}, fmt.Errorf("validating custom currency overage accrued result: %w", err)
				}
				if !handlerResult.TotalFiatAmount.Equal(line.Totals.Total) {
					return AccrueInvoiceUsageResult{}, fmt.Errorf(
						"custom currency overage booked fiat amount does not match line total: %s != %s",
						handlerResult.TotalFiatAmount,
						line.Totals.Total,
					)
				}

				ledgerTransactionRef = handlerResult.TransactionGroup
			} else {
				var err error
				ledgerTransactionRef, err = s.handler.OnInvoiceUsageAccrued(ctx, flatfee.OnInvoiceUsageAccruedInput{
					Charge:        in.Charge,
					ServicePeriod: line.Period,
					BookedAt:      flatfee.UsageBookedAt(in.Charge.Intent.GetEffectivePaymentTerm(), line.Period),
					Totals:        line.Totals,
				})
				if err != nil {
					return AccrueInvoiceUsageResult{}, fmt.Errorf("on flat fee standard invoice usage accrued: %w", err)
				}
			}

			if ledgerTransactionRef.TransactionGroupID == "" {
				return AccrueInvoiceUsageResult{}, fmt.Errorf("no ledger transaction is returned for run %s", currentRun.ID.ID)
			}

			accruedUsage := invoicedusage.AccruedUsage{
				ServicePeriod:     line.Period,
				Totals:            line.Totals,
				LedgerTransaction: &ledgerTransactionRef,
			}

			accruedUsage, err := s.adapter.CreateInvoicedUsage(ctx, flatfee.CreateInvoicedUsageInput{
				RunID:         currentRun.ID,
				LineID:        line.ID,
				InvoiceID:     in.LineWithHeader.Invoice.ID,
				InvoicedUsage: accruedUsage,
			})
			if err != nil {
				return AccrueInvoiceUsageResult{}, fmt.Errorf("creating standard invoice accrued usage: %w", err)
			}

			result.AccruedUsage = &accruedUsage
			result.Run.AccruedUsage = &accruedUsage
		}

		return result, nil
	})
}

type CorrectAccruedUsageInput struct {
	Charge flatfee.Charge
	Run    flatfee.RealizationRun
}

func (i CorrectAccruedUsageInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		errs = append(errs, errors.New("settlement mode must be credit_then_invoice"))
	}

	if !i.Charge.Intent.GetCurrency().IsCustom() {
		errs = append(errs, errors.New("charge currency must be custom"))
	}

	if i.Run.Immutable {
		errs = append(errs, errors.New("cannot correct accrued usage for immutable run"))
	}

	if i.Run.AccruedUsage == nil {
		errs = append(errs, errors.New("run must have accrued usage"))
	}

	if i.Charge.Realizations.CurrentRun == nil {
		errs = append(errs, errors.New("charge has no current realization run"))
	} else if i.Charge.Realizations.CurrentRun.ID != i.Run.ID {
		errs = append(errs, fmt.Errorf(
			"run is not the charge's current realization run [current_run_id=%s run_id=%s]",
			i.Charge.Realizations.CurrentRun.ID.ID,
			i.Run.ID.ID,
		))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// CorrectAccruedUsage reverses a mutable custom-currency run's prepared gross
// overage and removes the transient preparation so the run can be prepared again.
func (s *Service) CorrectAccruedUsage(ctx context.Context, input CorrectAccruedUsageInput) (flatfee.RealizationRun, error) {
	if err := input.Validate(); err != nil {
		return flatfee.RealizationRun{}, err
	}

	correctionInput := flatfee.OnCustomCurrencyOverageAccruedCorrectionInput{
		Charge: input.Charge,
		Run:    input.Run,
	}
	if err := correctionInput.Validate(); err != nil {
		return flatfee.RealizationRun{}, fmt.Errorf("validate custom-currency overage accrual correction: %w", err)
	}
	if err := s.handler.OnCustomCurrencyOverageAccruedCorrection(ctx, correctionInput); err != nil {
		return flatfee.RealizationRun{}, fmt.Errorf("correct custom-currency overage accrual: %w", err)
	}

	if err := s.adapter.DeleteInvoicedUsage(ctx, input.Run.AccruedUsage.NamespacedID); err != nil {
		return flatfee.RealizationRun{}, fmt.Errorf("delete custom-currency overage preparation: %w", err)
	}

	input.Run.AccruedUsage = nil

	return input.Run, nil
}

func (s *Service) MarkInvoiceIssued(ctx context.Context, run flatfee.RealizationRun) (flatfee.RealizationRun, error) {
	runBase, err := s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
		ID:        run.ID,
		Immutable: mo.Some(true),
	})
	if err != nil {
		return flatfee.RealizationRun{}, fmt.Errorf("updating issued realization run: %w", err)
	}

	run.RealizationRunBase = runBase

	return run, nil
}

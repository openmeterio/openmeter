package run

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type BookAccruedInvoiceUsageInput struct {
	Charge usagebased.Charge
	Run    usagebased.RealizationRun
	Line   billing.StandardLine
}

func (i BookAccruedInvoiceUsageInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if err := i.Run.Validate(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	if err := i.Line.Validate(); err != nil {
		return fmt.Errorf("line: %w", err)
	}

	if i.Run.LineID == nil {
		return fmt.Errorf("run %s has no linked standard line", i.Run.ID.ID)
	}

	if *i.Run.LineID != i.Line.ID {
		return fmt.Errorf("run %s already linked to a different line", i.Run.ID.ID)
	}

	if i.Run.InvoiceUsage != nil {
		return fmt.Errorf("run %s already has an invoice usage", i.Run.ID.ID)
	}

	if i.Run.NoFiatTransactionRequired && !i.Line.Totals.Total.IsZero() {
		return fmt.Errorf("run %s requires no fiat transaction but line total is non-zero", i.Run.ID.ID)
	}

	if !i.Run.NoFiatTransactionRequired && i.Line.Totals.Total.IsZero() {
		return fmt.Errorf("run %s has zero line total but requires a fiat transaction", i.Run.ID.ID)
	}

	return nil
}

type BookAccruedInvoiceUsageResult struct {
	Run          usagebased.RealizationRun
	InvoiceUsage *invoicedusage.AccruedUsage
}

func (s *Service) BookAccruedInvoiceUsage(ctx context.Context, in BookAccruedInvoiceUsageInput) (BookAccruedInvoiceUsageResult, error) {
	if err := in.Validate(); err != nil {
		return BookAccruedInvoiceUsageResult{}, err
	}

	if in.Run.NoFiatTransactionRequired {
		accruedUsage, err := s.adapter.CreateRunInvoicedUsage(ctx, in.Run.ID, invoicedusage.AccruedUsage{
			ServicePeriod: in.Line.Period,
			Totals:        in.Line.Totals,
		})
		if err != nil {
			return BookAccruedInvoiceUsageResult{}, fmt.Errorf("create invoiced usage for run %s: %w", in.Run.ID.ID, err)
		}

		in.Run.InvoiceUsage = &accruedUsage

		return BookAccruedInvoiceUsageResult{
			Run:          in.Run,
			InvoiceUsage: &accruedUsage,
		}, nil
	}

	var ledgerTransactionRef ledgertransaction.GroupReference
	if in.Charge.Intent.GetCurrency().IsCustom() {
		input := usagebased.OnCustomCurrencyOverageAccruedInput{
			Charge: in.Charge,
			Run:    in.Run,
		}
		if err := input.Validate(); err != nil {
			return BookAccruedInvoiceUsageResult{}, fmt.Errorf("validate on custom currency overage accrued input: %w", err)
		}

		result, err := s.handler.OnCustomCurrencyOverageAccrued(ctx, input)
		if err != nil {
			return BookAccruedInvoiceUsageResult{}, fmt.Errorf("on usage-based custom currency overage accrued: %w", err)
		}
		if err := result.Validate(); err != nil {
			return BookAccruedInvoiceUsageResult{}, fmt.Errorf("validate on custom currency overage accrued result: %w", err)
		}
		if !result.TotalFiatAmount.Equal(in.Line.Totals.Total) {
			return BookAccruedInvoiceUsageResult{}, fmt.Errorf(
				"custom currency overage booked fiat amount does not match line total: %s != %s",
				result.TotalFiatAmount,
				in.Line.Totals.Total,
			)
		}

		ledgerTransactionRef = result.TransactionGroup
	} else {
		input := usagebased.OnInvoiceUsageAccruedInput{
			Charge:        in.Charge,
			Run:           in.Run,
			ServicePeriod: in.Line.Period,
			BookedAt:      in.Line.Period.To,
			Amount:        in.Line.Totals.Total,
		}
		if err := input.Validate(); err != nil {
			return BookAccruedInvoiceUsageResult{}, fmt.Errorf("validate on invoice usage accrued input: %w", err)
		}

		var err error
		ledgerTransactionRef, err = s.handler.OnInvoiceUsageAccrued(ctx, input)
		if err != nil {
			return BookAccruedInvoiceUsageResult{}, fmt.Errorf("on usage-based invoice usage accrued: %w", err)
		}
	}

	if ledgerTransactionRef.TransactionGroupID == "" {
		return BookAccruedInvoiceUsageResult{}, fmt.Errorf("no ledger transaction is returned for run %s", in.Run.ID.ID)
	}

	accruedUsage, err := s.adapter.CreateRunInvoicedUsage(ctx, in.Run.ID, invoicedusage.AccruedUsage{
		ServicePeriod:     in.Line.Period,
		Totals:            in.Line.Totals,
		LedgerTransaction: &ledgerTransactionRef,
	})
	if err != nil {
		return BookAccruedInvoiceUsageResult{}, fmt.Errorf("create invoiced usage for run %s: %w", in.Run.ID.ID, err)
	}

	in.Run.InvoiceUsage = &accruedUsage

	return BookAccruedInvoiceUsageResult{
		Run:          in.Run,
		InvoiceUsage: &accruedUsage,
	}, nil
}

type CorrectAccruedUsageInput struct {
	Charge usagebased.Charge
	Run    usagebased.RealizationRun
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

	if i.Run.InvoiceUsage == nil {
		errs = append(errs, errors.New("run must have accrued usage"))
	}

	if i.Charge.State.CurrentRealizationRunID == nil {
		errs = append(errs, errors.New("charge has no current realization run"))
	} else if *i.Charge.State.CurrentRealizationRunID != i.Run.ID.ID {
		errs = append(errs, fmt.Errorf(
			"run is not the charge's current realization run [current_run_id=%s run_id=%s]",
			*i.Charge.State.CurrentRealizationRunID,
			i.Run.ID.ID,
		))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// CorrectAccruedUsage reverses a mutable custom-currency run's prepared gross
// overage and removes the transient preparation so the run can be prepared again.
func (s *Service) CorrectAccruedUsage(ctx context.Context, input CorrectAccruedUsageInput) (usagebased.RealizationRun, error) {
	if err := input.Validate(); err != nil {
		return usagebased.RealizationRun{}, err
	}

	correctionInput := usagebased.OnCustomCurrencyOverageAccruedCorrectionInput{
		Charge: input.Charge,
		Run:    input.Run,
	}
	if err := correctionInput.Validate(); err != nil {
		return usagebased.RealizationRun{}, fmt.Errorf("validate custom-currency overage accrual correction: %w", err)
	}
	if err := s.handler.OnCustomCurrencyOverageAccruedCorrection(ctx, correctionInput); err != nil {
		return usagebased.RealizationRun{}, fmt.Errorf("correct custom-currency overage accrual: %w", err)
	}

	if err := s.adapter.DeleteRunInvoicedUsage(ctx, input.Run.InvoiceUsage.NamespacedID); err != nil {
		return usagebased.RealizationRun{}, fmt.Errorf("delete custom-currency overage preparation: %w", err)
	}

	input.Run.InvoiceUsage = nil

	return input.Run, nil
}

func (s *Service) MarkInvoiceIssued(ctx context.Context, run usagebased.RealizationRun) (usagebased.RealizationRun, error) {
	runBase, err := s.adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:        run.ID,
		Immutable: mo.Some(true),
	})
	if err != nil {
		return usagebased.RealizationRun{}, fmt.Errorf("updating issued realization run: %w", err)
	}

	run.RealizationRunBase = runBase

	return run, nil
}

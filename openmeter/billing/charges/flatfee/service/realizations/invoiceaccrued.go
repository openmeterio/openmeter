package realizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
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

		if !line.Totals.Total.IsZero() {
			ledgerTransactionRef, err := s.handler.OnInvoiceUsageAccrued(ctx, flatfee.OnInvoiceUsageAccruedInput{
				Charge:        in.Charge,
				ServicePeriod: line.Period,
				Totals:        line.Totals,
			})
			if err != nil {
				return AccrueInvoiceUsageResult{}, fmt.Errorf("on flat fee standard invoice usage accrued: %w", err)
			}

			accruedUsage := invoicedusage.AccruedUsage{
				ServicePeriod:     line.Period,
				Totals:            line.Totals,
				LedgerTransaction: &ledgerTransactionRef,
			}

			accruedUsage, err = s.adapter.CreateInvoicedUsage(ctx, flatfee.CreateInvoicedUsageInput{
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

		runBase, err := s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:        currentRun.ID,
			Immutable: mo.Some(true),
		})
		if err != nil {
			return AccrueInvoiceUsageResult{}, fmt.Errorf("updating standard invoice run: %w", err)
		}

		result.Run.RealizationRunBase = runBase

		return result, nil
	})
}

package run

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
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

	if in.Line.Totals.Total.IsZero() {
		return BookAccruedInvoiceUsageResult{
			Run: in.Run,
		}, nil
	}

	input := usagebased.OnInvoiceUsageAccruedInput{
		Charge:        in.Charge,
		Run:           in.Run,
		ServicePeriod: in.Line.Period.ToClosedPeriod(),
		Amount:        in.Line.Totals.Total,
	}

	if err := input.Validate(); err != nil {
		return BookAccruedInvoiceUsageResult{}, fmt.Errorf("validate on invoice usage accrued input: %w", err)
	}

	ledgerTransactionRef, err := s.handler.OnInvoiceUsageAccrued(ctx, input)
	if err != nil {
		return BookAccruedInvoiceUsageResult{}, fmt.Errorf("on usage-based invoice usage accrued: %w", err)
	}

	if ledgerTransactionRef.TransactionGroupID == "" {
		return BookAccruedInvoiceUsageResult{}, fmt.Errorf("no ledger transaction is returned for run %s", in.Run.ID.ID)
	}

	accruedUsage, err := s.adapter.CreateRunInvoicedUsage(ctx, in.Run.ID, invoicedusage.AccruedUsage{
		LineID:            in.Run.LineID,
		ServicePeriod:     in.Line.Period.ToClosedPeriod(),
		Mutable:           false,
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

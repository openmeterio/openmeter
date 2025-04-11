package billing

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
)

type InvoiceLineWithInvoiceBase struct {
	Line    *Line
	Invoice InvoiceBase
}

func (i InvoiceLineWithInvoiceBase) Clone() InvoiceLineWithInvoiceBase {
	return InvoiceLineWithInvoiceBase{
		Line:    i.Line.Clone(),
		Invoice: i.Invoice,
	}
}

type InvoiceLineProgressiveHierarchy struct {
	Root     InvoiceLineWithInvoiceBase
	Children []InvoiceLineWithInvoiceBase
}

func (h *InvoiceLineProgressiveHierarchy) Clone() InvoiceLineProgressiveHierarchy {
	return InvoiceLineProgressiveHierarchy{
		Root: h.Root.Clone(),
		Children: lo.Map(h.Children, func(child InvoiceLineWithInvoiceBase, _ int) InvoiceLineWithInvoiceBase {
			return child.Clone()
		}),
	}
}

type SumNetAmountInput struct {
	PeriodEndLTE   time.Time
	IncludeCharges bool
}

// SumNetAmount returns the sum of the net amount (pre-tax) of the progressive billed line and its children
// containing the values for all lines whose period's end is <= in.UpTo and are not deleted or not part of
// an invoice that has been deleted.
func (h *InvoiceLineProgressiveHierarchy) SumNetAmount(in SumNetAmountInput) alpacadecimal.Decimal {
	netAmount := alpacadecimal.Zero

	_ = h.ForEachChild(ForEachChildInput{
		PeriodEndLTE: in.PeriodEndLTE,
		Callback: func(child InvoiceLineWithInvoiceBase) error {
			netAmount = netAmount.Add(child.Line.Totals.Amount)

			if in.IncludeCharges {
				netAmount = netAmount.Add(child.Line.Totals.ChargesTotal)
			}

			return nil
		},
	})

	return netAmount
}

type ForEachChildInput struct {
	PeriodEndLTE time.Time
	Callback     func(child InvoiceLineWithInvoiceBase) error
}

func (h *InvoiceLineProgressiveHierarchy) ForEachChild(in ForEachChildInput) error {
	for _, child := range h.Children {
		// The line is not in scope
		if !in.PeriodEndLTE.IsZero() && child.Line.Period.End.After(in.PeriodEndLTE) {
			continue
		}

		// The line is deleted
		if child.Line.DeletedAt != nil {
			continue
		}

		// The invoice is deleted
		if child.Invoice.DeletedAt != nil || child.Invoice.Status == InvoiceStatusDeleted {
			continue
		}

		if err := in.Callback(child); err != nil {
			return err
		}
	}

	return nil
}

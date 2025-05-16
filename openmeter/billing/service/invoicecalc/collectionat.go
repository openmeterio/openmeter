package invoicecalc

import (
	"errors"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func GatheringInvoiceCollectionAt(i *billing.Invoice, _ CalculatorDependencies) error {
	i.CollectionAt = nil

	if !i.Lines.IsPresent() {
		return errors.New("lines must be expanded")
	}

	for _, line := range i.Lines.OrEmpty() {
		if i.CollectionAt == nil {
			i.CollectionAt = lo.ToPtr(line.InvoiceAt)
			continue
		}

		if i.CollectionAt.After(line.InvoiceAt) {
			i.CollectionAt = lo.ToPtr(line.InvoiceAt)
		}
	}

	return nil
}

func StandardInvoiceCollectionAt(i *billing.Invoice, _ CalculatorDependencies) error {
	if !i.Lines.IsPresent() {
		return errors.New("lines must be expanded")
	}

	maxInvoiceAt := time.Time{}
	for _, line := range i.Lines.OrEmpty() {
		if line.Type != billing.InvoiceLineTypeUsageBased {
			// No collection required for non-usage based lines
			continue
		}

		if line.InvoiceAt.After(maxInvoiceAt) {
			maxInvoiceAt = line.InvoiceAt
		}
	}

	if maxInvoiceAt.IsZero() {
		i.CollectionAt = lo.ToPtr(i.CreatedAt)
	} else {
		collectionAt, _ := i.Workflow.Config.Collection.Interval.AddTo(maxInvoiceAt)

		// Given:
		// - we might be late creating the invoice, collectionAt might be before the invoice was created.
		// - DraftUntil is calculated based on collectionAt
		//
		// We might end up with not having a draft period ignoring the user intent, if we allow collectionAt to be
		// before the invoice was created.
		//
		// To avoid this, we adjust collectionAt to be not before the invoice was created, which results in a 0 long
		// collection period, but preserves the user intent to have a draft period.
		if collectionAt.Before(i.CreatedAt) {
			collectionAt = i.CreatedAt
		}

		i.CollectionAt = lo.ToPtr(collectionAt)
	}

	return nil
}

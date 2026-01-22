package invoicecalc

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func GatheringInvoiceCollectionAt(i *billing.StandardInvoice) error {
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

func StandardInvoiceCollectionAt(i *billing.StandardInvoice) error {
	if !i.Lines.IsPresent() {
		return errors.New("lines must be expanded")
	}

	maxInvoiceAt := time.Time{}
	for _, line := range i.Lines.OrEmpty() {
		if !line.DependsOnMeteredQuantity() {
			// No collection required for non-usage based lines
			continue
		}

		if line.InvoiceAt.After(maxInvoiceAt) {
			maxInvoiceAt = line.InvoiceAt
		}
	}

	// By default we can collect as soon as all lines are closed (which is driven by subscription sync)
	// (i.CreatedAt stubs in for current timestamp clock.Now(), effectively backdating for processing latency)
	collectionAt, _ := lo.Coalesce(maxInvoiceAt, i.CreatedAt)

	switch i.Workflow.Config.Collection.Alignment {
	case billing.AlignmentKindSubscription:
		// This is the trivial case
	case billing.AlignmentKindAnchored:
		// Let's calculate the next first day of month after the invoice was created
		startOfMonth := time.Date(collectionAt.Year(), collectionAt.Month(), 1, 0, 0, 0, 0, collectionAt.Location()) // FIXME: where should we get the Location from?
		collectionAt = startOfMonth.AddDate(0, 1, 0)
	default:
		return fmt.Errorf("unsupported collection alignment: %s", i.Workflow.Config.Collection.Alignment)
	}

	if collectionAt.IsZero() {
		return errors.New("failed to calculate default collection time")
	}

	// If we have an intended collection period, we should try to honor that
	if i.Workflow.Config.Collection.Interval.IsPositive() && !maxInvoiceAt.IsZero() {
		collectionAt, _ = i.Workflow.Config.Collection.Interval.AddTo(collectionAt)
	}

	// We push out collectionAt until invoice creation so we're always able to honor the user intent to have a draft period
	// if collectionAt.Before(i.CreatedAt) {
	// 	collectionAt = i.CreatedAt
	// }

	i.CollectionAt = lo.ToPtr(collectionAt)

	return nil
}

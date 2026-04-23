package invoicecalc

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func GatheringInvoiceCollectionAt(i *billing.GatheringInvoice, deps GatheringInvoiceCalculatorDependencies) error {
	i.NextCollectionAt = nil

	if !i.Lines.IsPresent() {
		return errors.New("lines must be expanded")
	}

	nonDeletedLines := i.Lines.NonDeletedLines()

	// Cannot determine next collection at without any non-deleted lines
	if len(nonDeletedLines) == 0 {
		return nil
	}

	minInvoiceAt := lo.MinBy(nonDeletedLines, func(a, b billing.GatheringLine) bool {
		return a.InvoiceAt.Before(b.InvoiceAt)
	})

	nextCollectionAt, err := calculateGatheringInvoiceNextCollectionAt(deps.Collection, minInvoiceAt.InvoiceAt)
	if err != nil {
		return fmt.Errorf("calculating next collection at: %w", err)
	}

	i.NextCollectionAt = lo.ToPtr(nextCollectionAt)

	return nil
}

func calculateGatheringInvoiceNextCollectionAt(collectionConfig billing.CollectionConfig, minInvoiceAt time.Time) (time.Time, error) {
	if err := collectionConfig.Validate(); err != nil {
		return time.Time{}, fmt.Errorf("invalid collection config: %w", err)
	}

	if minInvoiceAt.IsZero() {
		// Cannot determine next collection at without a min invoice at
		return time.Time{}, fmt.Errorf("cannot determine next collection at without a min invoice at")
	}

	if collectionConfig.Alignment == billing.AlignmentKindSubscription {
		return minInvoiceAt, nil
	}

	if collectionConfig.AnchoredAlignmentDetail == nil {
		return time.Time{}, errors.New("anchored alignment detail is required, when alignment is anchored")
	}

	recurrence, err := timeutil.NewRecurrenceFromISODuration(collectionConfig.AnchoredAlignmentDetail.Interval, collectionConfig.AnchoredAlignmentDetail.Anchor)
	if err != nil {
		return time.Time{}, fmt.Errorf("creating anchored alignment recurrence: %w", err)
	}

	next, err := recurrence.NextAfter(minInvoiceAt, timeutil.Inclusive)
	if err != nil {
		return time.Time{}, fmt.Errorf("resolving anchored alignment recurrence: %w", err)
	}

	return next, nil
}

func StandardInvoiceCollectionAt(i *billing.StandardInvoice) error {
	i.CollectionAt = nil

	if !i.Lines.IsPresent() {
		return errors.New("lines must be expanded")
	}

	collectableLines := lo.Filter(i.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) bool {
		return line.DeletedAt == nil && line.DependsOnMeteredQuantity()
	})

	if len(collectableLines) == 0 {
		return nil
	}

	maxInvoiceAt := lo.MaxBy(collectableLines, func(a, b *billing.StandardLine) bool {
		return a.InvoiceAt.After(b.InvoiceAt)
	})

	collectionAt := maxInvoiceAt.InvoiceAt

	// If we have an intended collection period, we should try to honor that
	if i.Workflow.Config.Collection.Interval.IsPositive() {
		collectionAt, _ = i.Workflow.Config.Collection.Interval.AddTo(collectionAt)
	}

	i.CollectionAt = lo.ToPtr(collectionAt)

	return nil
}

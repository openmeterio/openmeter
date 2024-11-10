package lineservice

import (
	"context"
	"time"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

var _ Line = feeLine{}

type feeLine struct {
	lineBase
}

func (l feeLine) PrepareForCreate(context.Context) (Line, error) {
	return l, nil
}

func (l feeLine) CanBeInvoicedAsOf(_ context.Context, t time.Time) (*billingentity.Period, error) {
	if !t.Before(l.line.InvoiceAt) {
		return &l.line.Period, nil
	}

	return nil, nil
}

func (l feeLine) SnapshotQuantity(context.Context, *billingentity.Invoice) (*snapshotQuantityResult, error) {
	return &snapshotQuantityResult{
		Line: l,
	}, nil
}

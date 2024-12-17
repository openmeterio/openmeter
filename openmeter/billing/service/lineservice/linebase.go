package lineservice

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type UpdateInput struct {
	ParentLine  mo.Option[*billing.Line]
	PeriodStart time.Time
	PeriodEnd   time.Time
	InvoiceAt   time.Time
	Status      billing.InvoiceLineStatus

	// PreventChildChanges is used to prevent any child changes to the line by the adapter.
	PreventChildChanges bool

	ResetChildUniqueReferenceID bool
}

func (i UpdateInput) apply(line *billing.Line) {
	if !i.PeriodStart.IsZero() {
		line.Period.Start = i.PeriodStart
	}

	if !i.PeriodEnd.IsZero() {
		line.Period.End = i.PeriodEnd
	}

	if !i.InvoiceAt.IsZero() {
		line.InvoiceAt = i.InvoiceAt
	}

	if i.Status != "" {
		line.Status = i.Status
	}

	if i.PreventChildChanges {
		line.Children = billing.LineChildren{}
	}

	if i.ResetChildUniqueReferenceID {
		line.ChildUniqueReferenceID = nil
	}
}

type SplitResult struct {
	PreSplitAtLine  Line
	PostSplitAtLine Line
}

type LineBase interface {
	ToEntity() *billing.Line
	ID() string
	InvoiceID() string
	Currency() currencyx.Code
	Period() billing.Period
	Status() billing.InvoiceLineStatus
	HasParent() bool
	// IsLastInPeriod returns true if the line is the last line in the period that is going to be invoiced.
	IsLastInPeriod() bool
	IsDeleted() bool
	IsSplit() bool

	CloneForCreate(in UpdateInput) Line
	Update(in UpdateInput) Line
	Save(context.Context) (Line, error)

	// Split splits a line into two lines at the given time.
	// The strategy is that we will have a line with status InvoiceLineStatusSplit and two child
	// lines with status InvoiceLineStatusValid.
	//
	// To make algorithms easier, upon next split, we will not create an imbalanced tree, but rather attach
	// the new split line to the existing parent line.
	Split(ctx context.Context, at time.Time) (SplitResult, error)

	Service() *Service
	ResetTotals()
}

var _ LineBase = (*lineBase)(nil)

type lineBase struct {
	line     *billing.Line
	service  *Service
	currency currencyx.Calculator
}

func (l lineBase) ToEntity() *billing.Line {
	return l.line
}

func (l lineBase) ID() string {
	return l.line.ID
}

func (l lineBase) InvoiceID() string {
	return l.line.InvoiceID
}

func (l lineBase) Currency() currencyx.Code {
	return l.line.Currency
}

func (l lineBase) Period() billing.Period {
	return l.line.Period
}

func (l lineBase) Status() billing.InvoiceLineStatus {
	return l.line.Status
}

func (l lineBase) HasParent() bool {
	return l.line.ParentLineID != nil
}

func (l lineBase) Validate(ctx context.Context, invoice *billing.Invoice) error {
	if l.line.Currency != invoice.Currency || l.line.Currency == "" {
		return billing.ValidationError{
			Err: billing.ErrInvoiceLineCurrencyMismatch,
		}
	}

	return nil
}

func (l lineBase) IsLastInPeriod() bool {
	return (l.line.Status == billing.InvoiceLineStatusValid && // We only care about valid lines
		(l.line.ParentLineID == nil || // Either we haven't split the line
			l.line.Period.End.Equal(l.line.ParentLine.Period.End))) // Or we have split the line and this is the last split
}

func (l lineBase) IsFirstInPeriod() bool {
	return (l.line.Status == billing.InvoiceLineStatusValid && // We only care about valid lines
		(l.line.ParentLineID == nil || // Either we haven't split the line
			l.line.Period.Start.Equal(l.line.ParentLine.Period.Start))) // Or we have split the line and this is the last split
}

func (l lineBase) IsDeleted() bool {
	return l.line.DeletedAt != nil
}

func (l lineBase) IsSplit() bool {
	return l.line.Status == billing.InvoiceLineStatusSplit
}

func (l lineBase) Save(ctx context.Context) (Line, error) {
	lines, err := l.service.BillingAdapter.UpsertInvoiceLines(ctx,
		billing.UpsertInvoiceLinesAdapterInput{
			Namespace: l.line.Namespace,
			Lines:     []*billing.Line{l.line},
		})
	if err != nil {
		return nil, fmt.Errorf("updating invoice line: %w", err)
	}

	return l.service.FromEntity(lines[0])
}

func (l lineBase) Service() *Service {
	return l.service
}

func (l lineBase) CloneForCreate(in UpdateInput) Line {
	outEntity := l.line.CloneWithoutDependencies()
	outEntity.ID = ""
	outEntity.CreatedAt = time.Time{}
	outEntity.UpdatedAt = time.Time{}

	out, _ := l.service.FromEntity(outEntity)

	return out.Update(in)
}

func (l lineBase) Update(in UpdateInput) Line {
	return l.update(in)
}

func (l lineBase) update(in UpdateInput) Line {
	in.apply(l.line)

	if in.ParentLine.IsPresent() {
		parentLine := in.ParentLine.OrEmpty()
		// Let's update the parent line
		if parentLine != nil {
			l.line.ParentLineID = lo.ToPtr(parentLine.ID)
			l.line.ParentLine = parentLine
		} else {
			l.line.ParentLineID = nil
			l.line.ParentLine = nil
		}
	}

	// Let's ignore the error here as we don't allow for any type updates
	svc, _ := l.service.FromEntity(l.line)

	return svc
}

// TODO[later]: We should rely on UpsertInvoiceLines and do this in bulk.
func (l lineBase) Split(ctx context.Context, splitAt time.Time) (SplitResult, error) {
	// We only split valid lines; split etc. lines are not supported
	if l.line.Status != billing.InvoiceLineStatusValid {
		return SplitResult{}, fmt.Errorf("line[%s]: line is not valid", l.line.ID)
	}

	if !l.line.Period.Contains(splitAt) {
		return SplitResult{}, fmt.Errorf("line[%s]: splitAt is not within the line period", l.line.ID)
	}

	if !l.HasParent() {
		parentLine, err := l.Update(UpdateInput{
			Status:              billing.InvoiceLineStatusSplit,
			PreventChildChanges: true,
		}).Save(ctx)
		if err != nil {
			return SplitResult{}, fmt.Errorf("saving parent line: %w", err)
		}

		// Let's create the child lines
		preSplitAtLine := l.CloneForCreate(UpdateInput{
			ParentLine:                  mo.Some(parentLine.ToEntity()),
			Status:                      billing.InvoiceLineStatusValid,
			PeriodEnd:                   splitAt,
			InvoiceAt:                   splitAt,
			ResetChildUniqueReferenceID: true,
		})

		postSplitAtLine := l.CloneForCreate(UpdateInput{
			ParentLine:                  mo.Some(parentLine.ToEntity()),
			Status:                      billing.InvoiceLineStatusValid,
			PeriodStart:                 splitAt,
			ResetChildUniqueReferenceID: true,
		})

		splitLines, err := l.service.UpsertLines(ctx, l.line.Namespace, preSplitAtLine, postSplitAtLine)
		if err != nil {
			return SplitResult{}, fmt.Errorf("creating split lines: %w", err)
		}

		return SplitResult{
			PreSplitAtLine:  splitLines[0],
			PostSplitAtLine: splitLines[1],
		}, nil
	}

	// We have alredy split the line once, we just need to create a new line and update the existing line
	postSplitAtLine, err := l.CloneForCreate(UpdateInput{
		Status:                      billing.InvoiceLineStatusValid,
		PeriodStart:                 splitAt,
		ParentLine:                  mo.Some(l.line.ParentLine),
		ResetChildUniqueReferenceID: true,
	}).Save(ctx)
	if err != nil {
		return SplitResult{}, fmt.Errorf("creating split lines: %w", err)
	}

	preSplitAtLine, err := l.Update(UpdateInput{
		PeriodEnd:                   splitAt,
		InvoiceAt:                   splitAt,
		ResetChildUniqueReferenceID: true,
	}).Save(ctx)
	if err != nil {
		return SplitResult{}, fmt.Errorf("updating parent line: %w", err)
	}

	return SplitResult{
		PreSplitAtLine:  preSplitAtLine,
		PostSplitAtLine: postSplitAtLine,
	}, nil
}

func (l lineBase) ResetTotals() {
	l.line.Totals = billing.Totals{}
}

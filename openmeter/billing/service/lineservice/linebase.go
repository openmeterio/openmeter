package lineservice

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type UpdateInput struct {
	ParentLine  *Line
	PeriodStart time.Time
	PeriodEnd   time.Time
	InvoiceAt   time.Time
	Status      billingentity.InvoiceLineStatus
}

func (i UpdateInput) apply(line *billingentity.Line) {
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
}

type SplitResult struct {
	PreSplitAtLine  Line
	PostSplitAtLine Line
}

type LineBase interface {
	ToEntity() billingentity.Line
	ID() string
	InvoiceID() string
	Currency() currencyx.Code
	Period() billingentity.Period
	Status() billingentity.InvoiceLineStatus
	HasParent() bool

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
}

var _ LineBase = (*lineBase)(nil)

type lineBase struct {
	line    billingentity.Line
	service *Service
}

func (l lineBase) ToEntity() billingentity.Line {
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

func (l lineBase) Period() billingentity.Period {
	return l.line.Period
}

func (l lineBase) Status() billingentity.InvoiceLineStatus {
	return l.line.Status
}

func (l lineBase) HasParent() bool {
	return l.line.ParentLineID != nil
}

func (l lineBase) Validate(ctx context.Context, invoice *billingentity.Invoice) error {
	if l.line.Currency != invoice.Currency || l.line.Currency == "" {
		return billingentity.ValidationError{
			Err: billingentity.ErrInvoiceLineCurrencyMismatch,
		}
	}

	return nil
}

func (l lineBase) Save(ctx context.Context) (Line, error) {
	line, err := l.service.BillingAdapter.UpdateInvoiceLine(ctx, billing.UpdateInvoiceLineAdapterInput(l.line))
	if err != nil {
		return nil, fmt.Errorf("updating invoice line: %w", err)
	}

	return l.service.FromEntity(line)
}

func (l lineBase) Service() *Service {
	return l.service
}

func (l lineBase) CloneForCreate(in UpdateInput) Line {
	l.line.ID = ""
	l.line.CreatedAt = time.Time{}
	l.line.UpdatedAt = time.Time{}

	return l.update(in)
}

func (l lineBase) Update(in UpdateInput) Line {
	return l.update(in)
}

func (l lineBase) update(in UpdateInput) Line {
	in.apply(&l.line)

	// Let's update the parent line
	if in.ParentLine != nil {
		newParentLine := *in.ParentLine
		if newParentLine == nil {
			l.line.ParentLineID = nil
			l.line.ParentLine = nil
		} else {
			l.line.ParentLineID = lo.ToPtr(newParentLine.ID())
			l.line.ParentLine = lo.ToPtr(newParentLine.ToEntity())
		}
	}

	// Let's ignore the error here as we don't allow for any type updates
	svc, _ := l.service.FromEntity(l.line)

	return svc
}

func (l lineBase) Split(ctx context.Context, splitAt time.Time) (SplitResult, error) {
	// We only split valid lines; split etc. lines are not supported
	if l.line.Status != billingentity.InvoiceLineStatusValid {
		return SplitResult{}, fmt.Errorf("line[%s]: line is not valid", l.line.ID)
	}

	if !l.line.Period.Contains(splitAt) {
		return SplitResult{}, fmt.Errorf("line[%s]: splitAt is not within the line period", l.line.ID)
	}

	if !l.HasParent() {
		parentLine, err := l.Update(UpdateInput{
			Status: billingentity.InvoiceLineStatusSplit,
		}).Save(ctx)
		if err != nil {
			return SplitResult{}, fmt.Errorf("saving parent line: %w", err)
		}

		// Let's create the child lines
		preSplitAtLine := l.CloneForCreate(UpdateInput{
			ParentLine: &parentLine,
			Status:     billingentity.InvoiceLineStatusValid,
			PeriodEnd:  splitAt,
			InvoiceAt:  splitAt,
		})

		postSplitAtLine := l.CloneForCreate(UpdateInput{
			ParentLine:  &parentLine,
			Status:      billingentity.InvoiceLineStatusValid,
			PeriodStart: splitAt,
		})

		splitLines, err := l.service.CreateLines(ctx, preSplitAtLine, postSplitAtLine)
		if err != nil {
			return SplitResult{}, fmt.Errorf("creating split lines: %w", err)
		}

		return SplitResult{
			PreSplitAtLine:  splitLines[0],
			PostSplitAtLine: splitLines[1],
		}, nil
	}

	// We have alredy split the line once, we just need to create a new line and update the existing line
	postSplitAtLine := l.CloneForCreate(UpdateInput{
		Status:      billingentity.InvoiceLineStatusValid,
		PeriodStart: splitAt,
	})

	createdLines, err := l.service.CreateLines(ctx, postSplitAtLine)
	if err != nil {
		return SplitResult{}, fmt.Errorf("creating split lines: %w", err)
	}

	postSplitAtLine = createdLines[0]

	preSplitAtLine, err := l.Update(UpdateInput{
		PeriodEnd: splitAt,
		InvoiceAt: splitAt,
	}).Save(ctx)
	if err != nil {
		return SplitResult{}, fmt.Errorf("updating parent line: %w", err)
	}

	return SplitResult{
		PreSplitAtLine:  preSplitAtLine,
		PostSplitAtLine: postSplitAtLine,
	}, nil
}

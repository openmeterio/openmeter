package lineservice

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type UpdateInput struct {
	SplitLineGroupID string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	InvoiceAt        time.Time
	Status           billing.InvoiceLineStatus

	ResetChildUniqueReferenceID bool
}

type SplitResult struct {
	PreSplitAtLine  Line
	PostSplitAtLine Line
}

type LineBase interface {
	ToEntity() *billing.Line
	ID() string
	InvoiceAt() time.Time
	InvoiceID() string
	Currency() currencyx.Code
	Period() billing.Period
	// IsLastInPeriod returns true if the line is the last line in the period that is going to be invoiced.
	IsLastInPeriod() bool
	IsDeleted() bool
	IsSplitLineGroupMember() bool

	CloneForCreate(in UpdateInput) Line
	Update(in UpdateInput) Line
	Save(context.Context) (Line, error)
	Delete(context.Context) error
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

func (l lineBase) InvoiceAt() time.Time {
	return l.line.InvoiceAt
}

func (l lineBase) Currency() currencyx.Code {
	return l.line.Currency
}

func (l lineBase) Period() billing.Period {
	return l.line.Period
}

func (l lineBase) IsSplitLineGroupMember() bool {
	return l.line.SplitLineGroupID != nil
}

func (l lineBase) Validate(ctx context.Context, invoice *billing.Invoice) error {
	if l.line.Currency != invoice.Currency || l.line.Currency == "" {
		return billing.ValidationError{
			Err: billing.ErrInvoiceLineCurrencyMismatch,
		}
	}

	// Expanding the split lines are mandatory for the lineservice to work properly.
	if l.line.SplitLineGroupID != nil && l.line.SplitLineHierarchy == nil {
		return billing.ValidationError{
			Err: fmt.Errorf("split line group[%s] has no expanded hierarchy, while being part of a split line group", *l.line.SplitLineGroupID),
		}
	}

	return nil
}

func (l lineBase) IsLastInPeriod() bool {
	if l.line.SplitLineGroupID == nil {
		return true
	}

	if l.line.SplitLineHierarchy.Group.ServicePeriod.End.Equal(l.line.Period.End) {
		return true
	}

	return false
}

func (l lineBase) IsFirstInPeriod() bool {
	if l.line.SplitLineGroupID == nil {
		return true
	}

	if l.line.SplitLineHierarchy.Group.ServicePeriod.Start.Equal(l.line.Period.Start) {
		return true
	}

	return false
}

func (l lineBase) IsDeleted() bool {
	return l.line.DeletedAt != nil
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

func (l lineBase) Delete(ctx context.Context) error {
	l.line.DeletedAt = lo.ToPtr(clock.Now())

	_, err := l.Save(ctx)

	return err
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
	// TODO[later]: Either we should clone and update the clone or we should not return the Line as if that's a new
	// object.

	if !in.PeriodStart.IsZero() {
		l.line.Period.Start = in.PeriodStart
	}

	if !in.PeriodEnd.IsZero() {
		l.line.Period.End = in.PeriodEnd
	}

	if !in.InvoiceAt.IsZero() {
		l.line.InvoiceAt = in.InvoiceAt
	}

	if in.SplitLineGroupID != "" {
		l.line.SplitLineGroupID = lo.ToPtr(in.SplitLineGroupID)
	}

	if in.ResetChildUniqueReferenceID {
		l.line.ChildUniqueReferenceID = nil
	}

	// Let's ignore the error here as we don't allow for any type updates
	svc, _ := l.service.FromEntity(l.line)

	return svc
}

// TODO[later]: We should rely on UpsertInvoiceLines and do this in bulk.
func (l lineBase) Split(ctx context.Context, splitAt time.Time) (SplitResult, error) {
	if !l.line.Period.Contains(splitAt) {
		return SplitResult{}, fmt.Errorf("line[%s]: splitAt is not within the line period", l.line.ID)
	}

	var splitLineGroupID string
	if !l.IsSplitLineGroupMember() {
		splitLineGroup, err := l.service.BillingAdapter.CreateSplitLineGroup(ctx, billing.CreateSplitLineGroupAdapterInput{
			Namespace: l.line.Namespace,

			SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
				Name:        l.line.Name,
				Description: l.line.Description,

				ServicePeriod:     l.line.Period,
				RatecardDiscounts: l.line.RateCardDiscounts,
				TaxConfig:         l.line.TaxConfig,
			},

			UniqueReferenceID: l.line.ChildUniqueReferenceID,

			Currency: l.line.Currency,

			Price:      l.line.UsageBased.Price,
			FeatureKey: lo.EmptyableToPtr(l.line.UsageBased.FeatureKey),

			Subscription: l.line.Subscription,
		})
		if err != nil {
			return SplitResult{}, fmt.Errorf("creating split line group: %w", err)
		}

		splitLineGroupID = splitLineGroup.ID
	} else {
		splitLineGroupID = *l.line.SplitLineGroupID
	}

	result := SplitResult{}

	// We have alredy split the line once, we just need to create a new line and update the existing line
	postSplitAtLine := l.CloneForCreate(UpdateInput{
		PeriodStart:                 splitAt,
		SplitLineGroupID:            splitLineGroupID,
		ResetChildUniqueReferenceID: true,
	})
	if !postSplitAtLine.IsPeriodEmptyConsideringTruncations() {
		postSplitAtLine, err := postSplitAtLine.Save(ctx)
		if err != nil {
			return SplitResult{}, fmt.Errorf("saving post split line: %w", err)
		}

		result.PostSplitAtLine = postSplitAtLine
	}

	preSplitAtLine := l.Update(UpdateInput{
		PeriodEnd:                   splitAt,
		InvoiceAt:                   splitAt,
		SplitLineGroupID:            splitLineGroupID,
		ResetChildUniqueReferenceID: true,
	})

	if !preSplitAtLine.IsPeriodEmptyConsideringTruncations() {
		preSplitAtLine, err := preSplitAtLine.Save(ctx)
		if err != nil {
			return SplitResult{}, fmt.Errorf("saving pre split line: %w", err)
		}

		result.PreSplitAtLine = preSplitAtLine
	} else {
		if err := preSplitAtLine.Delete(ctx); err != nil {
			return SplitResult{}, fmt.Errorf("deleting pre split line: %w", err)
		}
	}

	return result, nil
}

func (l lineBase) ResetTotals() {
	l.line.Totals = billing.Totals{}
}

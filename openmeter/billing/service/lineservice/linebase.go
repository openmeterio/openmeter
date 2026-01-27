package lineservice

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
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
	ToEntity() *billing.StandardLine
	ID() string
	InvoiceAt() time.Time
	InvoiceID() string
	Currency() currencyx.Code
	Period() billing.Period
	// IsLastInPeriod returns true if the line is the last line in the period that is going to be invoiced.
	IsLastInPeriod() bool
	IsDeleted() bool

	ResetTotals()
}

var _ LineBase = (*lineBase)(nil)

type lineBase struct {
	line          *billing.StandardLine
	featureMeters billing.FeatureMeters
	currency      currencyx.Calculator
}

func (l lineBase) ToEntity() *billing.StandardLine {
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

func (l lineBase) IsLastInPeriod() bool {
	if l.line.SplitLineGroupID == nil {
		return true
	}

	if l.line.SplitLineHierarchy == nil {
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

	if l.line.SplitLineHierarchy == nil {
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

func (l lineBase) ResetTotals() {
	l.line.Totals = billing.Totals{}
}

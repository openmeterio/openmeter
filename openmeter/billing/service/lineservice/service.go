package lineservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type Service struct {
	Config
}

type Config struct {
	StreamingConnector streaming.Connector
}

func (c Config) Validate() error {
	if c.StreamingConnector == nil {
		return fmt.Errorf("streaming connector is required")
	}

	return nil
}

func New(in Config) (*Service, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		Config: in,
	}, nil
}

func (s *Service) FromEntity(line *billing.StandardLine, featureMeters billing.FeatureMeters) (Line, error) {
	currencyCalc, err := line.Currency.Calculator()
	if err != nil {
		return nil, fmt.Errorf("creating currency calculator: %w", err)
	}

	base := lineBase{
		service:       s,
		featureMeters: featureMeters,
		line:          line,
		currency:      currencyCalc,
	}

	if line.UsageBased.Price.Type() == productcatalog.FlatPriceType {
		return &ubpFlatFeeLine{
			lineBase: base,
		}, nil
	}

	return &usageBasedLine{
		lineBase: base,
	}, nil
}

func (s *Service) FromEntities(line []*billing.StandardLine, featureMeters billing.FeatureMeters) (Lines, error) {
	return slicesx.MapWithErr(line, func(l *billing.StandardLine) (Line, error) {
		return s.FromEntity(l, featureMeters)
	})
}

// UpdateTotalsFromDetailedLines is a helper method to update the totals of a line from its detailed lines.
func (s *Service) UpdateTotalsFromDetailedLines(line *billing.StandardLine) error {
	// Calculate the line totals
	for idx, detailedLine := range line.DetailedLines {
		if detailedLine.DeletedAt != nil {
			continue
		}

		totals, err := calculateDetailedLineTotals(detailedLine)
		if err != nil {
			return fmt.Errorf("updating totals for line[%s]: %w", line.ID, err)
		}

		line.DetailedLines[idx].Totals = totals
	}

	// WARNING: Even if tempting to add discounts etc. here to the totals, we should always keep the logic as is.
	// The usageBasedLine will never be syncorinzed directly to stripe or other apps, only the detailed lines.
	//
	// Given that the external systems will have their own logic for calculating the totals, we cannot expect
	// any custom logic implemented here to be carried over to the external systems.

	// UBP line's value is the sum of all the children
	res := billing.Totals{}

	res = res.Add(lo.Map(line.DetailedLines, func(l billing.DetailedLine, _ int) billing.Totals {
		// Deleted lines are not contributing to the totals
		if l.DeletedAt != nil {
			return billing.Totals{}
		}

		return l.Totals
	})...)

	line.Totals = res

	return nil
}

type InvoicingCapabilityQueryInput struct {
	AsOf               time.Time
	ProgressiveBilling bool
}

type (
	CanBeInvoicedAsOfInput     = InvoicingCapabilityQueryInput
	ResolveBillablePeriodInput = InvoicingCapabilityQueryInput
)

type Line interface {
	LineBase

	Service() *Service

	// IsPeriodEmptyConsideringTruncations returns true if the line has an empty period. This is different from Period.IsEmpty() as
	// this method does any truncation for usage based lines.
	IsPeriodEmptyConsideringTruncations() bool

	Validate(context.Context, *billing.StandardInvoice) error
	CanBeInvoicedAsOf(context.Context, CanBeInvoicedAsOfInput) (*billing.Period, error)
	SnapshotQuantity(ctx context.Context, customer billing.InvoiceCustomer) error
	CalculateDetailedLines() error
	PrepareForCreate(context.Context) (Line, error)
	UpdateTotals() error
}

type Lines []Line

func (s Lines) ValidateForInvoice(ctx context.Context, invoice *billing.StandardInvoice) error {
	return errors.Join(lo.Map(s, func(line Line, idx int) error {
		if line == nil {
			return fmt.Errorf("line[%d] is nil", idx)
		}

		if err := line.Validate(ctx, invoice); err != nil {
			id := line.ID()
			if id == "" {
				id = fmt.Sprintf("line[%d]", idx)
			}

			return fmt.Errorf("line[%s]: %w", id, err)
		}

		return nil
	})...)
}

func (s Lines) ToEntities() []*billing.StandardLine {
	return lo.Map(s, func(service Line, _ int) *billing.StandardLine {
		return service.ToEntity()
	})
}

type LineWithBillablePeriod struct {
	Line
	BillablePeriod billing.Period
}

func (s Lines) ResolveBillablePeriod(ctx context.Context, in ResolveBillablePeriodInput) ([]LineWithBillablePeriod, error) {
	out := make([]LineWithBillablePeriod, 0, len(s))
	for _, lineSrv := range s {
		billablePeriod, err := lineSrv.CanBeInvoicedAsOf(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("checking if line can be invoiced: %w", err)
		}

		if billablePeriod != nil {
			out = append(out, LineWithBillablePeriod{
				Line:           lineSrv,
				BillablePeriod: *billablePeriod,
			})
		}
	}

	if len(out) == 0 {
		// We haven't requested explicit empty invoice, so we should have some pending lines
		return nil, billing.ValidationError{
			Err: billing.ErrInvoiceCreateNoLines,
		}
	}

	return out, nil
}

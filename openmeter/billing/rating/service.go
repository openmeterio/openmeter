package rating

import (
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type Service interface {
	ResolveBillablePeriod(in ResolveBillablePeriodInput) (billing.IsLineBillableAsOfResult, error)
	GenerateDetailedLines(in StandardLineAccessor, opts ...GenerateDetailedLinesOption) (GenerateDetailedLinesResult, error)
}

type GenerateDetailedLinesOptions struct {
	IgnoreMinimumCommitment bool
	DisableCreditsMutator   bool
}

type GenerateDetailedLinesOption func(*GenerateDetailedLinesOptions)

func NewGenerateDetailedLinesOptions(opts ...GenerateDetailedLinesOption) GenerateDetailedLinesOptions {
	var out GenerateDetailedLinesOptions

	for _, opt := range opts {
		opt(&out)
	}

	return out
}

func WithMinimumCommitmentIgnored() GenerateDetailedLinesOption {
	return func(o *GenerateDetailedLinesOptions) {
		o.IgnoreMinimumCommitment = true
	}
}

func WithCreditsMutatorDisabled() GenerateDetailedLinesOption {
	return func(o *GenerateDetailedLinesOptions) {
		o.DisableCreditsMutator = true
	}
}

type Usage struct {
	Quantity              alpacadecimal.Decimal
	PreLinePeriodQuantity alpacadecimal.Decimal
}

type GenerateDetailedLinesResult struct {
	DetailedLines DetailedLines
	// FinalUsage is the final usage of the line after all the discounts have been applied
	FinalUsage *Usage
	// FinalStandardLineDiscounts is the final standard line discounts for the line after all the discounts have been applied
	FinalStandardLineDiscounts billing.StandardLineDiscounts

	// Totals is the totals of the line after all the calculations have been applied
	Totals totals.Totals
}

type ResolveBillablePeriodInput struct {
	AsOf               time.Time
	ProgressiveBilling bool
	Line               GatheringLineAccessor
	Feature            *feature.Feature
	Meter              *meter.Meter
}

func (i ResolveBillablePeriodInput) Validate() error {
	var errs []error

	if i.Line == nil {
		errs = append(errs, errors.New("line is required"))
	} else {
		price := i.Line.GetPrice()
		if price == nil {
			errs = append(errs, errors.New("line price is required"))
		} else if price.Type() != productcatalog.FlatPriceType {
			if i.Feature == nil {
				errs = append(errs, errors.New("feature is required for metered lines"))
			}

			if i.Meter == nil {
				errs = append(errs, errors.New("meter is required for metered lines"))
			}
		}
	}

	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	return errors.Join(errs...)
}

package rating

import (
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Service interface {
	ResolveBillablePeriod(in ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error)
	GenerateDetailedLines(in StandardLineAccessor, opts ...GenerateDetailedLinesOption) (GenerateDetailedLinesResult, error)
}

type GenerateDetailedLinesOptions struct {
	IgnoreMinimumCommitment bool
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
	FeatureMeters      feature.FeatureMeters
}

func (i ResolveBillablePeriodInput) Validate() error {
	if i.Line == nil {
		return fmt.Errorf("line is required")
	}

	if i.FeatureMeters == nil {
		return fmt.Errorf("feature meters are required")
	}

	if i.AsOf.IsZero() {
		return fmt.Errorf("as of is required")
	}

	return nil
}

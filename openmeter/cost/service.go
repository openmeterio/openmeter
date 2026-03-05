package cost

import (
	"context"
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Service provides cost computation for features.
type Service interface {
	QueryFeatureCost(ctx context.Context, input QueryFeatureCostInput) (*CostQueryResult, error)
}

// QueryFeatureCostInput is the input for querying the cost of a feature.
type QueryFeatureCostInput struct {
	Namespace   string
	FeatureID   string
	QueryParams streaming.QueryParams
}

func (i QueryFeatureCostInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.FeatureID == "" {
		errs = append(errs, errors.New("feature ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// CostQueryRow represents a single row in a feature cost query result.
type CostQueryRow struct {
	Usage       alpacadecimal.Decimal
	Cost        *alpacadecimal.Decimal
	Currency    currencyx.Code
	Detail      string
	Subject     *string
	CustomerID  *string
	GroupBy     map[string]*string
	WindowStart time.Time
	WindowEnd   time.Time
}

// CostQueryResult is the result of querying feature costs.
type CostQueryResult struct {
	Currency currencyx.Code
	Rows     []CostQueryRow
}

// ResolvedUnitCost is the result of resolving a feature's per-unit cost.
type ResolvedUnitCost struct {
	// Amount is the resolved per-unit cost.
	Amount alpacadecimal.Decimal
	// Currency is the currency code (always "USD" for now).
	Currency currencyx.Code
}

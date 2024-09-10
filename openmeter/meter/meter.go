package meter

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

// TODO: move types here and add aliases to the models package

type Meter = models.Meter

type MeterAggregation = models.MeterAggregation

const (
	MeterAggregationSum         = models.MeterAggregationSum
	MeterAggregationCount       = models.MeterAggregationCount
	MeterAggregationAvg         = models.MeterAggregationAvg
	MeterAggregationMin         = models.MeterAggregationMin
	MeterAggregationMax         = models.MeterAggregationMax
	MeterAggregationUniqueCount = models.MeterAggregationUniqueCount
)

type WindowSize = models.WindowSize

const (
	WindowSizeMinute = models.WindowSizeMinute
	WindowSizeHour   = models.WindowSizeHour
	WindowSizeDay    = models.WindowSizeDay
)

// Repository is an interface to the meter store.
type Repository interface {
	// ListAllMeters returns a list of meters.
	ListAllMeters(ctx context.Context) ([]Meter, error)

	// ListMeters returns a list of meters for the given namespace.
	ListMeters(ctx context.Context, namespace string) ([]Meter, error)

	// GetMeterByIDOrSlug returns a meter from the meter store by ID or slug.
	GetMeterByIDOrSlug(ctx context.Context, namespace string, idOrSlug string) (Meter, error)
}

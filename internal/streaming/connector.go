package streaming

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ListEventsParams struct {
	From  *time.Time
	To    *time.Time
	Limit int
}

type Connector interface {
	ListEvents(ctx context.Context, namespace string, params ListEventsParams) ([]api.IngestedEvent, error)
	CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error
	DeleteMeter(ctx context.Context, namespace string, meterSlug string) error
	QueryMeter(ctx context.Context, namespace string, meterSlug string, params *QueryParams) ([]models.MeterQueryRow, error)
	ListMeterSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error)
	// Add more methods as needed ...
}

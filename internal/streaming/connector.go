package streaming

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type GetValuesParams struct {
	From       *time.Time
	To         *time.Time
	Subject    *string
	WindowSize *models.WindowSize
}

type Connector interface {
	CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error
	DeleteMeter(ctx context.Context, namespace string, meterSlug string) error
	QueryMeter(ctx context.Context, namespace string, meterSlug string, params *GetValuesParams) ([]*models.MeterValue, error)
	// Add more methods as needed ...
}

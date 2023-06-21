package streaming

import (
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
	Close() error
	Init(meter *models.Meter) error
	GetValues(meter *models.Meter, params *GetValuesParams) ([]*models.MeterValue, error)
	Healthcheck() error
	// Add more methods as needed ...
}

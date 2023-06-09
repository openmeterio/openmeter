package streaming

import (
	"time"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/internal/models"
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
	Publish(event event.Event) error
	GetValues(meter *models.Meter, params *GetValuesParams) ([]*models.MeterValue, error)
	// Add more methods as needed ...
}

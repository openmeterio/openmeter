package streaming

import (
	"context"
	"errors"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/pkg/models"
)

type ListEventsParams struct {
	Limit int
}

type QueryParams struct {
	From           *time.Time
	To             *time.Time
	Subject        []string
	GroupBySubject bool
	GroupBy        []string
	Aggregation    models.MeterAggregation
	WindowSize     *models.WindowSize
}

func (p *QueryParams) Validate() error {
	if p.From != nil && p.To != nil && p.From.After(*p.To) {
		return errors.New("from must be before to")
	}

	if p.WindowSize != nil {
		windowDuration := p.WindowSize.Duration()
		if p.From != nil && p.From.Truncate(windowDuration) != *p.From {
			return errors.New("from must be aligned to window size")
		}
		if p.To != nil && p.To.Truncate(windowDuration) != *p.To {
			return errors.New("to must be aligned to window size")
		}
	}

	return nil
}

type QueryResult struct {
	WindowSize *models.WindowSize
	Values     []*models.MeterValue
}

type Connector interface {
	ListEvents(ctx context.Context, namespace string, params ListEventsParams) ([]event.Event, error)
	CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error
	DeleteMeter(ctx context.Context, namespace string, meterSlug string) error
	QueryMeter(ctx context.Context, namespace string, meterSlug string, params *QueryParams) (*QueryResult, error)
	ListMeterSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error)
	// Add more methods as needed ...
}

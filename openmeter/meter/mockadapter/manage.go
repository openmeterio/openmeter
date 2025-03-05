package adapter

import (
	"context"

	"github.com/oklog/ulid/v2"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// CreateMeter creates a new meter.
func (a manageAdapter) CreateMeter(ctx context.Context, input meterpkg.CreateMeterInput) (meterpkg.Meter, error) {
	meter := meterpkg.Meter{
		ID:            ulid.Make().String(),
		Name:          input.Name,
		Description:   input.Description,
		Key:           input.Key,
		Aggregation:   input.Aggregation,
		EventType:     input.EventType,
		ValueProperty: input.ValueProperty,
		GroupBy:       input.GroupBy,
		WindowSize:    meterpkg.WindowSizeMinute,
	}

	a.adapter.meters = append(a.adapter.meters, meter)

	return meter, nil
}

// DeleteMeter deletes a meter.
func (a manageAdapter) DeleteMeter(ctx context.Context, input meterpkg.DeleteMeterInput) error {
	for i, m := range a.adapter.meters {
		if m.Namespace != input.Namespace {
			continue
		}

		if m.ID == input.IDOrSlug || m.Key == input.IDOrSlug {
			a.adapter.meters = append(a.adapter.meters[:i], a.adapter.meters[i+1:]...)
			return nil
		}
	}

	return meterpkg.NewMeterNotFoundError(input.IDOrSlug)
}

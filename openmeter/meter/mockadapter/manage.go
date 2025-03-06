package adapter

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreateMeter creates a new meter.
func (a manageAdapter) CreateMeter(ctx context.Context, input meterpkg.CreateMeterInput) (meterpkg.Meter, error) {
	meter := meterpkg.Meter{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: input.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name:        input.Name,
			Description: input.Description,
		},

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

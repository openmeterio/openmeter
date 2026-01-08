package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// RegisterPreUpdateMeterHook registers a pre-update meter hook.
func (a manageAdapter) RegisterPreUpdateMeterHook(hook meterpkg.PreUpdateMeterHook) error {
	return models.NewGenericNotImplementedError(
		fmt.Errorf("pre-update meter hook is not implemented in mock adapter"),
	)
}

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
		EventFrom:     input.EventFrom,
		ValueProperty: input.ValueProperty,
		GroupBy:       input.GroupBy,
	}

	a.adapter.meters = append(a.adapter.meters, meter)

	return meter, nil
}

// UpdateMeter updates a meter.
func (a manageAdapter) UpdateMeter(ctx context.Context, input meterpkg.UpdateMeterInput) (meterpkg.Meter, error) {
	currentMeter, err := a.GetMeterByIDOrSlug(ctx, meterpkg.GetMeterInput{
		Namespace: input.ID.Namespace,
		IDOrSlug:  input.ID.ID,
	})
	if err != nil {
		return meterpkg.Meter{}, err
	}

	meter := meterpkg.Meter{
		ManagedResource: models.ManagedResource{
			// Immutable fields
			ID:              currentMeter.ID,
			NamespacedModel: currentMeter.NamespacedModel,
			ManagedModel: models.ManagedModel{
				CreatedAt: currentMeter.CreatedAt,
				UpdatedAt: time.Now(),
			},
			// Mutable fields
			Name:        input.Name,
			Description: input.Description,
		},
		// Immutable fields
		Key:           currentMeter.Key,
		Aggregation:   currentMeter.Aggregation,
		EventType:     currentMeter.EventType,
		ValueProperty: currentMeter.ValueProperty,
		// Mutable fields
		EventFrom: currentMeter.EventFrom,
		GroupBy:   input.GroupBy,
	}

	for i, m := range a.adapter.meters {
		if m.Namespace != input.ID.Namespace {
			continue
		}

		if m.ID == input.ID.ID || m.Key == currentMeter.Key {
			a.adapter.meters[i] = meter
			return meter, nil
		}
	}

	return meter, meterpkg.NewMeterNotFoundError(currentMeter.Key)
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

func (a manageAdapter) UpdateTableEngine(ctx context.Context, meter meterpkg.Meter) error {
	return models.NewGenericNotImplementedError(
		fmt.Errorf("update table engine is not implemented in mock adapter"),
	)
}

func (a manageAdapter) DeleteTableEngine(ctx context.Context, meter meterpkg.Meter) error {
	return models.NewGenericNotImplementedError(
		fmt.Errorf("delete table engine is not implemented in mock adapter"),
	)
}

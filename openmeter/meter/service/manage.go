package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ meter.ManageService = (*ManageService)(nil)

type ManageService struct {
	meter.Service
	preUpdateHooks        []meter.PreUpdateMeterHook
	adapter               *adapter.Adapter
	publisher             eventbus.Publisher
	entitlementRepository entitlement.EntitlementRepo
	featureRepository     feature.FeatureRepo
	namespaceManager      *namespace.Manager
	streamingConnector    streaming.Connector
}

func NewManage(
	adapter *adapter.Adapter,
	publisher eventbus.Publisher,
	entitlementRepository entitlement.EntitlementRepo,
	featureRepository feature.FeatureRepo,
	namespaceManager *namespace.Manager,
	streamingConnector streaming.Connector,
) *ManageService {
	return &ManageService{
		Service:               New(adapter),
		adapter:               adapter,
		publisher:             publisher,
		entitlementRepository: entitlementRepository,
		featureRepository:     featureRepository,
		namespaceManager:      namespaceManager,
		streamingConnector:    streamingConnector,
	}
}

// RegisterPreUpdateMeterHook registers a hook to be called before updating a meter.
func (s *ManageService) RegisterPreUpdateMeterHook(hook meter.PreUpdateMeterHook) error {
	s.preUpdateHooks = append(s.preUpdateHooks, hook)
	return nil
}

// CreateMeter creates a meter
func (s *ManageService) CreateMeter(ctx context.Context, input meter.CreateMeterInput) (meter.Meter, error) {
	// Create the meter
	createdMeter, err := s.adapter.CreateMeter(ctx, input)
	if err != nil {
		return createdMeter, err
	}

	// TODO: remove this once we are sure that the namespace is created at signup
	err = s.namespaceManager.CreateNamespace(ctx, input.Namespace)
	if err != nil {
		return createdMeter, fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create the meter in the streaming connector
	err = s.streamingConnector.CreateMeter(ctx, input.Namespace, createdMeter)
	if err != nil {
		return createdMeter, fmt.Errorf("failed to create meter in streaming connector: %w", err)
	}

	// Publish the meter created event
	meterCreatedEvent := meter.NewMeterCreateEvent(ctx, &createdMeter)
	if err := s.publisher.Publish(ctx, meterCreatedEvent); err != nil {
		return createdMeter, fmt.Errorf("failed to publish meter created event: %w", err)
	}

	return createdMeter, nil
}

// DeleteMeter deletes a meter
func (s *ManageService) DeleteMeter(ctx context.Context, input meter.DeleteMeterInput) error {
	// Get the meter
	getMeter, err := s.GetMeterByIDOrSlug(ctx, meter.GetMeterInput(input))
	if err != nil {
		return err
	}

	// Check if the meter is already deleted
	if getMeter.DeletedAt != nil {
		return meter.NewMeterNotFoundError(getMeter.Key)
	}

	// Check if the meter has active features
	hasFeatures, err := s.featureRepository.HasActiveFeatureForMeter(ctx, input.Namespace, getMeter.Key)
	if err != nil {
		return fmt.Errorf("failed to check if meter has features: %w", err)
	}

	if hasFeatures {
		return models.NewGenericConflictError(
			fmt.Errorf("meter has active features and cannot be deleted"),
		)
	}

	// Check if the meter has active entitlements
	hasEntitlements, err := s.entitlementRepository.HasEntitlementForMeter(ctx, getMeter.Namespace, getMeter.Key)
	if err != nil {
		return fmt.Errorf("failed to check if meter has entitlements: %w", err)
	}

	if hasEntitlements {
		return models.NewGenericConflictError(
			fmt.Errorf("meter has active entitlements and cannot be deleted"),
		)
	}

	// Delete the meter
	err = s.adapter.DeleteMeter(ctx, getMeter)
	if err != nil {
		return err
	}

	// Get the deleted meter
	deletedMeter, err := s.GetMeterByIDOrSlug(ctx, meter.GetMeterInput(input))
	if err != nil {
		return err
	}

	// Delete the meter in the streaming connector
	err = s.streamingConnector.DeleteMeter(ctx, input.Namespace, deletedMeter)
	if err != nil {
		return fmt.Errorf("failed to delete meter in streaming connector: %w", err)
	}

	// Publish the meter deleted event
	meterDeletedEvent := meter.NewMeterDeleteEvent(ctx, &deletedMeter)
	if err := s.publisher.Publish(ctx, meterDeletedEvent); err != nil {
		return fmt.Errorf("failed to publish meter deleted event: %w", err)
	}

	return nil
}

// UpdateMeter updates a meter
func (s *ManageService) UpdateMeter(ctx context.Context, input meter.UpdateMeterInput) (meter.Meter, error) {
	// Get the meter by ID
	currentMeter, err := s.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: input.ID.Namespace,
		IDOrSlug:  input.ID.ID,
	})
	if err != nil {
		return meter.Meter{}, err
	}

	if err := input.Validate(currentMeter.ValueProperty); err != nil {
		return meter.Meter{}, models.NewGenericValidationError(err)
	}

	// Collect group by changes
	var groupByToDelete []string

	for key := range currentMeter.GroupBy {
		if _, ok := input.GroupBy[key]; !ok {
			groupByToDelete = append(groupByToDelete, key)
		}
	}

	// FIXME: use foreign keys after we migrate Feature reference on meter id
	// Check if features are compatible with the new group by values
	// We only need to check deleted group bys because only those can be incompatible
	if len(groupByToDelete) > 0 {
		// List features depending on the meter
		features, err := s.featureRepository.ListFeatures(ctx, feature.ListFeaturesParams{
			Namespace:  input.ID.Namespace,
			MeterSlugs: []string{currentMeter.Key},
		})
		if err != nil {
			return meter.Meter{}, fmt.Errorf("failed to list features for meter: %w", err)
		}

		// Check if the features are compatible with the new group by values
		for _, feature := range features.Items {
			for _, groupBy := range groupByToDelete {
				if _, ok := feature.MeterGroupByFilters[groupBy]; ok {
					return meter.Meter{}, models.NewGenericConflictError(
						fmt.Errorf("meter group by: %s cannot be dropped because it is used by feature: %s", groupBy, feature.Key),
					)
				}
			}
		}
	}

	// Run pre-update hooks
	for _, hook := range s.preUpdateHooks {
		if err := hook(ctx, input); err != nil {
			return meter.Meter{}, err
		}
	}

	// Update the meter
	updatedMeter, err := s.adapter.UpdateMeter(ctx, input)
	if err != nil {
		return meter.Meter{}, err
	}

	// Update the meter in the streaming connector
	err = s.streamingConnector.UpdateMeter(ctx, input.ID.Namespace, updatedMeter)
	if err != nil {
		return updatedMeter, fmt.Errorf("failed to update meter in streaming connector: %w", err)
	}

	// Publish the meter updated event
	meterUpdatedEvent := meter.NewMeterUpdateEvent(ctx, &updatedMeter)
	if err := s.publisher.Publish(ctx, meterUpdatedEvent); err != nil {
		return updatedMeter, fmt.Errorf("failed to publish meter updated event: %w", err)
	}

	return updatedMeter, nil
}

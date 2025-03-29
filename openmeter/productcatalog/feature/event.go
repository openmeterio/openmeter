package feature

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
)

const (
	FeatureEventSubsystem   metadata.EventSubsystem = "feature"
	FeatureCreateEventName  metadata.EventName      = "feature.created"
	FeatureUpdateEventName  metadata.EventName      = "feature.updated"
	FeatureArchiveEventName metadata.EventName      = "feature.archived"
)

// NewFeatureCreateEvent creates a new feature create event
func NewFeatureCreateEvent(ctx context.Context, feature *Feature) FeatureCreateEvent {
	return FeatureCreateEvent{
		Feature: feature,
		UserID:  session.GetSessionUserID(ctx),
	}
}

// FeatureCreateEvent is an event that is emitted when a feature is created
type FeatureCreateEvent struct {
	Feature *Feature `json:"feature"`
	UserID  *string  `json:"userId,omitempty"`
}

func (e FeatureCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: FeatureEventSubsystem,
		Name:      FeatureCreateEventName,
		Version:   "v1",
	})
}

func (e FeatureCreateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Feature.Namespace, metadata.EntityFeature, e.Feature.ID)

	return metadata.EventMetadata{
		ID:      e.Feature.ID,
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Feature.CreatedAt,
	}
}

func (e FeatureCreateEvent) Validate() error {
	var errs []error

	if e.Feature == nil {
		return fmt.Errorf("feature is required")
	}

	if err := e.Feature.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("feature: %w", err))
	}

	return errors.Join(errs...)
}

// NewFeatureArchiveEvent creates a new feature delete event
func NewFeatureArchiveEvent(ctx context.Context, feature *Feature) FeatureArchiveEvent {
	return FeatureArchiveEvent{
		Feature: feature,
		UserID:  session.GetSessionUserID(ctx),
	}
}

// FeatureArchiveEvent is an event that is emitted when a feature is archived
type FeatureArchiveEvent struct {
	Feature *Feature `json:"feature"`
	UserID  *string  `json:"userId,omitempty"`
}

func (e FeatureArchiveEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: FeatureEventSubsystem,
		Name:      FeatureArchiveEventName,
		Version:   "v1",
	})
}

func (e FeatureArchiveEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Feature.Namespace, metadata.EntityFeature, e.Feature.ID)

	return metadata.EventMetadata{
		ID:      e.Feature.ID,
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    *e.Feature.ArchivedAt,
	}
}

func (e FeatureArchiveEvent) Validate() error {
	var errs []error

	if e.Feature == nil {
		return fmt.Errorf("feature is required")
	}

	if e.Feature.ArchivedAt == nil {
		return fmt.Errorf("feature archived at is required")
	}

	if err := e.Feature.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("feature: %w", err))
	}

	return errors.Join(errs...)
}

package plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
)

const (
	PlanEventSubsystem   metadata.EventSubsystem = "plan"
	PlanCreateEventName  metadata.EventName      = "plan.created"
	PlanUpdateEventName  metadata.EventName      = "plan.updated"
	PlanDeleteEventName  metadata.EventName      = "plan.deleted"
	PlanPublishEventName metadata.EventName      = "plan.published"
	PlanArchiveEventName metadata.EventName      = "plan.archived"
)

// NewPlanCreateEvent creates a new plan create event
func NewPlanCreateEvent(ctx context.Context, plan *Plan) PlanCreateEvent {
	return PlanCreateEvent{
		Plan:   plan,
		UserID: session.GetSessionUserID(ctx),
	}
}

// PlanCreateEvent is an event that is emitted when a plan is created
type PlanCreateEvent struct {
	Plan   *Plan   `json:"plan"`
	UserID *string `json:"userId,omitempty"`
}

func (e PlanCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: PlanEventSubsystem,
		Name:      PlanCreateEventName,
		Version:   "v1",
	})
}

func (e PlanCreateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Plan.Namespace, metadata.EntityPlan, e.Plan.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Plan.CreatedAt,
	}
}

func (e PlanCreateEvent) Validate() error {
	var errs []error

	if e.Plan == nil {
		return fmt.Errorf("plan is required")
	}

	return errors.Join(errs...)
}

// NewPlanUpdateEvent creates a new plan update event
func NewPlanUpdateEvent(ctx context.Context, plan *Plan) PlanUpdateEvent {
	return PlanUpdateEvent{
		Plan:   plan,
		UserID: session.GetSessionUserID(ctx),
	}
}

// PlanUpdateEvent is an event that is emitted when a plan is updated
type PlanUpdateEvent struct {
	Plan   *Plan   `json:"plan"`
	UserID *string `json:"userId,omitempty"`
}

func (e PlanUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: PlanEventSubsystem,
		Name:      PlanUpdateEventName,
		Version:   "v1",
	})
}

func (e PlanUpdateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Plan.Namespace, metadata.EntityPlan, e.Plan.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Plan.UpdatedAt,
	}
}

func (e PlanUpdateEvent) Validate() error {
	var errs []error

	if e.Plan == nil {
		return fmt.Errorf("plan is required")
	}

	return errors.Join(errs...)
}

// NewPlanDeleteEvent creates a new plan delete event
func NewPlanDeleteEvent(ctx context.Context, plan *Plan) PlanDeleteEvent {
	return PlanDeleteEvent{
		Plan:   plan,
		UserID: session.GetSessionUserID(ctx),
	}
}

// PlanDeleteEvent is an event that is emitted when a plan is deleted
type PlanDeleteEvent struct {
	Plan   *Plan   `json:"plan"`
	UserID *string `json:"userId,omitempty"`
}

func (e PlanDeleteEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: PlanEventSubsystem,
		Name:      PlanDeleteEventName,
		Version:   "v1",
	})
}

func (e PlanDeleteEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Plan.Namespace, metadata.EntityPlan, e.Plan.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    *e.Plan.DeletedAt,
	}
}

func (e PlanDeleteEvent) Validate() error {
	var errs []error

	if e.Plan == nil {
		return fmt.Errorf("plan is required")
	}

	if e.Plan.DeletedAt == nil {
		return fmt.Errorf("plan deleted at is required")
	}

	return errors.Join(errs...)
}

// NewPlanPublishEvent creates a new plan publish event
func NewPlanPublishEvent(ctx context.Context, plan *Plan) PlanPublishEvent {
	return PlanPublishEvent{
		Plan:   plan,
		UserID: session.GetSessionUserID(ctx),
	}
}

// PlanPublishEvent is an event that is emitted when a plan is published
type PlanPublishEvent struct {
	Plan   *Plan   `json:"plan"`
	UserID *string `json:"userId,omitempty"`
}

func (e PlanPublishEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: PlanEventSubsystem,
		Name:      PlanPublishEventName,
		Version:   "v1",
	})
}

func (e PlanPublishEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Plan.Namespace, metadata.EntityPlan, e.Plan.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Plan.UpdatedAt,
	}
}

func (e PlanPublishEvent) Validate() error {
	var errs []error

	if e.Plan == nil {
		return fmt.Errorf("plan is required")
	}

	return errors.Join(errs...)
}

// NewPlanArchiveEvent creates a new plan archive event
func NewPlanArchiveEvent(ctx context.Context, plan *Plan) PlanArchiveEvent {
	return PlanArchiveEvent{
		Plan:   plan,
		UserID: session.GetSessionUserID(ctx),
	}
}

// PlanArchiveEvent is an event that is emitted when a plan is archived
type PlanArchiveEvent struct {
	Plan   *Plan   `json:"plan"`
	UserID *string `json:"userId,omitempty"`
}

func (e PlanArchiveEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: PlanEventSubsystem,
		Name:      PlanArchiveEventName,
		Version:   "v1",
	})
}

func (e PlanArchiveEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Plan.Namespace, metadata.EntityPlan, e.Plan.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Plan.UpdatedAt,
	}
}

func (e PlanArchiveEvent) Validate() error {
	var errs []error

	if e.Plan == nil {
		return fmt.Errorf("plan is required")
	}

	return errors.Join(errs...)
}

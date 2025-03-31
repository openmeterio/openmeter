package app

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
)

const (
	AppEventSubsystem  metadata.EventSubsystem = "app"
	AppCreateEventName metadata.EventName      = "app.created"
	AppUpdateEventName metadata.EventName      = "app.updated"
	AppDeleteEventName metadata.EventName      = "app.deleted"
)

// NewAppCreateEvent creates a new app create event
func NewAppCreateEvent(ctx context.Context, appBase AppBase) AppCreateEvent {
	return AppCreateEvent{
		AppBase: appBase,
		UserID:  session.GetSessionUserID(ctx),
	}
}

// AppCreateEvent is an event that is emitted when an app is created
type AppCreateEvent struct {
	AppBase AppBase `json:"appBase"`
	UserID  *string `json:"userId,omitempty"`
}

func (e AppCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppCreateEventName,
		Version:   "v1",
	})
}

func (e AppCreateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.AppBase.Namespace, metadata.EntityApp, e.AppBase.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.AppBase.CreatedAt,
	}
}

func (e AppCreateEvent) Validate() error {
	if e.AppBase.ID == "" {
		return fmt.Errorf("app base is required")
	}
	return nil
}

// NewAppUpdateEvent creates a new app update event
func NewAppUpdateEvent(ctx context.Context, appBase AppBase) AppUpdateEvent {
	return AppUpdateEvent{
		AppBase: appBase,
		UserID:  session.GetSessionUserID(ctx),
	}
}

// AppUpdateEvent is an event that is emitted when an app is updated
type AppUpdateEvent struct {
	AppBase AppBase `json:"appBase"`
	UserID  *string `json:"userId,omitempty"`
}

func (e AppUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppUpdateEventName,
		Version:   "v1",
	})
}

func (e AppUpdateEvent) EventMetadata() metadata.EventMetadata {
	appBase := e.AppBase.GetAppBase()
	resourcePath := metadata.ComposeResourcePath(appBase.Namespace, metadata.EntityApp, appBase.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    appBase.UpdatedAt,
	}
}

func (e AppUpdateEvent) Validate() error {
	if e.AppBase.ID == "" {
		return fmt.Errorf("app base is required")
	}

	return nil
}

// NewAppDeleteEvent creates a new app delete event
func NewAppDeleteEvent(ctx context.Context, appBase AppBase) AppDeleteEvent {
	return AppDeleteEvent{
		AppBase: appBase,
		UserID:  session.GetSessionUserID(ctx),
	}
}

// AppDeleteEvent is an event that is emitted when an app is deleted
type AppDeleteEvent struct {
	AppBase AppBase `json:"appBase"`
	UserID  *string `json:"userId,omitempty"`
}

func (e AppDeleteEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppDeleteEventName,
		Version:   "v1",
	})
}

func (e AppDeleteEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.AppBase.Namespace, metadata.EntityApp, e.AppBase.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    *e.AppBase.DeletedAt,
	}
}

func (e AppDeleteEvent) Validate() error {
	if e.AppBase.ID == "" {
		return fmt.Errorf("app base is required")
	}

	return nil
}

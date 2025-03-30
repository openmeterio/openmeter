package app

import (
	"context"
	"fmt"

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
func NewAppCreateEvent(ctx context.Context, app AppBase) AppCreateEvent {
	return AppCreateEvent{
		App:    app,
		UserID: session.GetSessionUserID(ctx),
	}
}

// AppCreateEvent is an event that is emitted when an app is created
type AppCreateEvent struct {
	App    AppBase `json:"app"`
	UserID *string `json:"userId,omitempty"`
}

func (e AppCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppCreateEventName,
		Version:   "v1",
	})
}

func (e AppCreateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.App.Namespace, metadata.EntityApp, e.App.ID)

	return metadata.EventMetadata{
		ID:      e.App.ID,
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.App.CreatedAt,
	}
}

func (e AppCreateEvent) Validate() error {
	if e.App.ID == "" {
		return fmt.Errorf("app is required")
	}
	return nil
}

// NewAppUpdateEvent creates a new app update event
func NewAppUpdateEvent(ctx context.Context, app App) AppUpdateEvent {
	return AppUpdateEvent{
		App:    app,
		UserID: session.GetSessionUserID(ctx),
	}
}

// AppUpdateEvent is an event that is emitted when an app is updated
type AppUpdateEvent struct {
	App    App     `json:"app"`
	UserID *string `json:"userId,omitempty"`
}

func (e AppUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppUpdateEventName,
		Version:   "v1",
	})
}

func (e AppUpdateEvent) EventMetadata() metadata.EventMetadata {
	appBase := e.App.GetAppBase()
	resourcePath := metadata.ComposeResourcePath(appBase.Namespace, metadata.EntityApp, appBase.ID)

	return metadata.EventMetadata{
		ID:      appBase.ID,
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    appBase.UpdatedAt,
	}
}

func (e AppUpdateEvent) Validate() error {
	if e.App == nil {
		return fmt.Errorf("app is required")
	}
	return nil
}

// NewAppDeleteEvent creates a new app delete event
func NewAppDeleteEvent(ctx context.Context, app App) AppDeleteEvent {
	return AppDeleteEvent{
		App:    app,
		UserID: session.GetSessionUserID(ctx),
	}
}

// AppDeleteEvent is an event that is emitted when an app is deleted
type AppDeleteEvent struct {
	App    App     `json:"app"`
	UserID *string `json:"userId,omitempty"`
}

func (e AppDeleteEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppDeleteEventName,
		Version:   "v1",
	})
}

func (e AppDeleteEvent) EventMetadata() metadata.EventMetadata {
	appBase := e.App.GetAppBase()
	resourcePath := metadata.ComposeResourcePath(appBase.Namespace, metadata.EntityApp, appBase.ID)

	return metadata.EventMetadata{
		ID:      appBase.ID,
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    *appBase.DeletedAt,
	}
}

func (e AppDeleteEvent) Validate() error {
	if e.App == nil {
		return fmt.Errorf("app is required")
	}
	appBase := e.App.GetAppBase()
	if appBase.DeletedAt == nil {
		return fmt.Errorf("app deleted at is required")
	}
	return nil
}

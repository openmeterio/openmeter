package notification

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Repository interface {
	ChannelRepository
	RuleRepository
	EventRepository
}

type ChannelRepository interface {
	ListChannels(ctx context.Context, params ListChannelsInput) (pagination.PagedResponse[Channel], error)
	CreateChannel(ctx context.Context, params CreateChannelInput) (*Channel, error)
	DeleteChannel(ctx context.Context, params DeleteChannelInput) error
	GetChannel(ctx context.Context, params GetChannelInput) (*Channel, error)
	UpdateChannel(ctx context.Context, params UpdateChannelInput) (*Channel, error)
}

type RuleRepository interface {
	ListRules(ctx context.Context, params ListRulesInput) (pagination.PagedResponse[Rule], error)
	CreateRule(ctx context.Context, params CreateRuleInput) (*Rule, error)
	DeleteRule(ctx context.Context, params DeleteRuleInput) error
	GetRule(ctx context.Context, params GetRuleInput) (*Rule, error)
	UpdateRule(ctx context.Context, params UpdateRuleInput) (*Rule, error)
}

type EventRepository interface {
	ListEvents(ctx context.Context, params ListEventsInput) (pagination.PagedResponse[Event], error)
	GetEvent(ctx context.Context, params GetEventInput) (*Event, error)
	CreateEvent(ctx context.Context, params CreateEventInput) (*Event, error)
	ListEventsDeliveryStatus(ctx context.Context, params ListEventsDeliveryStatusInput) (pagination.PagedResponse[EventDeliveryStatus], error)
	GetEventDeliveryStatus(ctx context.Context, params GetEventDeliveryStatusInput) (*EventDeliveryStatus, error)
	CreateEventDeliveryStatus(ctx context.Context, params CreateEventDeliveryStatusInput) (*EventDeliveryStatus, error)
}

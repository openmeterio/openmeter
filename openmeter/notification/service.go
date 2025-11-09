package notification

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

const (
	OrderByID        OrderBy = "id"
	OrderByType      OrderBy = "type"
	OrderByCreatedAt OrderBy = "createdAt"
	OrderByUpdatedAt OrderBy = "updatedAt"
)

type OrderBy string

type Service interface {
	FeatureService

	ChannelService
	RuleService
	EventService
}

type ChannelService interface {
	ListChannels(ctx context.Context, params ListChannelsInput) (ListChannelsResult, error)
	CreateChannel(ctx context.Context, params CreateChannelInput) (*Channel, error)
	DeleteChannel(ctx context.Context, params DeleteChannelInput) error
	GetChannel(ctx context.Context, params GetChannelInput) (*Channel, error)
	UpdateChannel(ctx context.Context, params UpdateChannelInput) (*Channel, error)
}

type RuleService interface {
	ListRules(ctx context.Context, params ListRulesInput) (ListRulesResult, error)
	CreateRule(ctx context.Context, params CreateRuleInput) (*Rule, error)
	DeleteRule(ctx context.Context, params DeleteRuleInput) error
	GetRule(ctx context.Context, params GetRuleInput) (*Rule, error)
	UpdateRule(ctx context.Context, params UpdateRuleInput) (*Rule, error)
}

type EventService interface {
	ListEvents(ctx context.Context, params ListEventsInput) (ListEventsResult, error)
	GetEvent(ctx context.Context, params GetEventInput) (*Event, error)
	CreateEvent(ctx context.Context, params CreateEventInput) (*Event, error)
	ResendEvent(ctx context.Context, params ResendEventInput) error
	ListEventsDeliveryStatus(ctx context.Context, params ListEventsDeliveryStatusInput) (ListEventsDeliveryStatusResult, error)
	GetEventDeliveryStatus(ctx context.Context, params GetEventDeliveryStatusInput) (*EventDeliveryStatus, error)
	UpdateEventDeliveryStatus(ctx context.Context, params UpdateEventDeliveryStatusInput) (*EventDeliveryStatus, error)
}

type FeatureService interface {
	ListFeature(ctx context.Context, namespace string, features ...string) ([]feature.Feature, error)
}

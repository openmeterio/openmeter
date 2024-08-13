package notification

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Repository interface {
	ChannelRepository
	RuleRepository
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

package notification

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Repository interface {
	ChannelRepository
}

type ChannelRepository interface {
	ListChannels(ctx context.Context, params ListChannelsInput) (pagination.PagedResponse[Channel], error)
	CreateChannel(ctx context.Context, params CreateChannelInput) (*Channel, error)
	DeleteChannel(ctx context.Context, params DeleteChannelInput) error
	GetChannel(ctx context.Context, params GetChannelInput) (*Channel, error)
	UpdateChannel(ctx context.Context, params UpdateChannelInput) (*Channel, error)
}

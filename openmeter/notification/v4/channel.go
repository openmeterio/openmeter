package v4

import "github.com/openmeterio/openmeter/pkg/models"

type ChannelType string

// Channel exposes the identity of a concrete channel type.
type Channel interface {
	ChannelType() ChannelType
}

// ChannelConfig defines validation shared by providers of a channel type.
type ChannelConfig interface {
	models.Validator

	isInternal()
}

// ChannelConfigUpdate validates only explicitly supplied configuration fields.
type ChannelConfigUpdate interface {
	models.Validator

	isChannelConfigUpdate()
}

type channel[C ChannelConfig] struct {
	// Name is the user provided name of the Channel.
	Name string `json:"name"`

	// Disabled defines whether the Channel is disabled or not.
	Disabled bool `json:"disabled"`

	// Config stores the actual Channel configuration specific to the Type.
	Config C `json:"config"`
}

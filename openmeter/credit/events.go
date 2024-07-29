package credit

import (
	"github.com/openmeterio/openmeter/internal/credit/grant"
)

const (
	EventSubsystem = grant.EventSubsystem
)

const (
	EventCreateGrant = grant.GrantCreatedEvent
	EventVoidGrant   = grant.GrantVoidedEvent
)

type (
	GrantCreatedEvent = grant.GrantCreatedEvent
	GrantVoidedEvent  = grant.GrantVoidedEvent
)

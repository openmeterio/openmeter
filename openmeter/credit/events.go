package credit

import (
	"github.com/openmeterio/openmeter/internal/credit/grant"
)

const (
	EventSubsystem = grant.EventSubsystem
)

type (
	GrantCreatedEvent = grant.CreatedEvent
	GrantVoidedEvent  = grant.VoidedEvent
)

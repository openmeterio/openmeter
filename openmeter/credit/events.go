package credit

import "github.com/openmeterio/openmeter/internal/credit"

const (
	EventSubsystem = credit.EventSubsystem
)

const (
	EventCreateGrant = credit.EventCreateGrant
	EventVoidGrant   = credit.EventVoidGrant
)

type (
	GrantCreatedEvent = credit.GrantCreatedEvent
	GrantVoidedEvent  = credit.GrantVoidedEvent
)

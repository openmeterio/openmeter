package credit

import (
	"github.com/openmeterio/openmeter/internal/event/spec"
)

const (
	EventSubsystemCredit spec.EventSubsystem = "credit"

	EventCreateGrant spec.EventName = "createGrant"
	EventVoidGrant   spec.EventName = "voidGrant"
)

type GrantEvent struct {
	Grant

	SubjectKey string `json:"subjectKey"`
	// Namespace from Grant cannot be used as it will never be serialized
	Namespace string `json:"namespace"`
}

type GrantCreatedEvent GrantEvent

var grantCreatedEventSpec = spec.EventTypeSpec{
	Subsystem:   EventSubsystemCredit,
	Name:        EventCreateGrant,
	SpecVersion: "1.0",
	Version:     "v1",
}

func (e GrantCreatedEvent) Spec() *spec.EventTypeSpec {
	return &grantCreatedEventSpec
}

type GrantVoidedEvent GrantEvent

var grantVoidedEventSpec = spec.EventTypeSpec{
	Subsystem:   EventSubsystemCredit,
	Name:        EventVoidGrant,
	SpecVersion: "1.0",
	Version:     "v1",
}

func (e GrantVoidedEvent) Spec() *spec.EventTypeSpec {
	return &grantVoidedEventSpec
}

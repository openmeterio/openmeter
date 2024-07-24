package schema

import (
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/event/types"
)

const (
	subjectKindGrant = "grant"
	subsystemCredit  = "credit"
)

type GrantEvent struct {
	credit.Grant

	SubjectKey string `json:"subjectKey"`
	// Namespace from credit.Grant cannot be used as it will never be serialized
	Namespace string `json:"namespace"`
}

type GrantCreatedEvent GrantEvent

var grantCreatedEventSpec = types.EventTypeSpec{
	Subsystem:   subsystemCredit,
	Name:        "createGrant",
	SpecVersion: "1.0",
	Version:     "v1",
	SubjectKind: subjectKindGrant,
}

func (e GrantCreatedEvent) Spec() *types.EventTypeSpec {
	return &grantCreatedEventSpec
}

type GrantVoidedEvent GrantEvent

var grantVoidedEventSpec = types.EventTypeSpec{
	Subsystem:   subsystemCredit,
	Name:        "voidGrant",
	SpecVersion: "1.0",
	Version:     "v1",
	SubjectKind: subjectKindGrant,
}

func (e GrantVoidedEvent) Spec() *types.EventTypeSpec {
	return &grantVoidedEventSpec
}

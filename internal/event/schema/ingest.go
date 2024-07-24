package schema

import (
	"github.com/openmeterio/openmeter/internal/event/types"
)

const (
	subsystemIngest = "ingest"
	subjectKeyKind  = "subjectKey"
)

type IngestEvent struct {
	Namespace string `json:"namespace"`
	Subject   string `json:"subject"`

	// MeterSlugs contain the list of slugs that are affected by the event. We
	// should not use meterIDs as they are not something present in the open source
	// version, thus any code that is in opensource should not rely on them.
	MeterSlugs []string `json:"meterSlugs"`
}

var ingestEventSpec = types.EventTypeSpec{
	Subsystem:   subsystemIngest,
	Name:        "ingestion",
	SpecVersion: "1.0",
	Version:     "v1",
	SubjectKind: subjectKeyKind,
}

func (i IngestEvent) Spec() *types.EventTypeSpec {
	return &ingestEventSpec
}

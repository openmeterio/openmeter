package snapshot

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type ValueOperationType string

const (
	ValueOperationUpdate ValueOperationType = "update"
	ValueOperationDelete ValueOperationType = "delete"
)

type EntitlementValue struct {
	// Balance Only available for metered entitlements. Metered entitlements are built around a balance calculation where feature usage is deducted from the issued grants. Balance represents the remaining balance of the entitlement, it's value never turns negative.
	Balance *float64 `json:"balance,omitempty"`

	// Config Only available for static entitlements. The JSON parsable config of the entitlement.
	Config *string `json:"config,omitempty"`

	// HasAccess Whether the subject has access to the feature. Shared across all entitlement types.
	HasAccess *bool `json:"hasAccess,omitempty"`

	// Overage Only available for metered entitlements. Overage represents the usage that wasn't covered by grants, e.g. if the subject had a total feature usage of 100 in the period but they were only granted 80, there would be 20 overage.
	Overage *float64 `json:"overage,omitempty"`

	// Usage Only available for metered entitlements. Returns the total feature usage in the current period.
	Usage *float64 `json:"usage,omitempty"`
}

type SnapshotEvent struct {
	Entitlement entitlement.Entitlement `json:"entitlement"`
	Namespace   models.NamespaceID      `json:"namespace"`
	Subject     models.Subject          `json:"subject"`
	Feature     productcatalog.Feature  `json:"feature"`
	// Operation is delete if the entitlement gets deleted, in that case the balance object is empty
	Operation ValueOperationType `json:"operation"`

	// CalculatedAt specifies when the balance calculation was performed. It can be used to verify
	// in edge-worker if the store already contains the required item.
	CalculatedAt *time.Time `json:"calculatedAt,omitempty"`

	Value              *EntitlementValue  `json:"value,omitempty"`
	CurrentUsagePeriod *recurrence.Period `json:"currentUsagePeriod,omitempty"`
}

var (
	_ marshaler.Event = SnapshotEvent{}

	snapshotEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: entitlement.EventSubsystem,
		Name:      "entitlement.snapshot",
		Version:   "v2",
	})
)

func (e SnapshotEvent) EventName() string {
	return snapshotEventName
}

func (e SnapshotEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntitySubjectKey, e.Subject.Key),
	}
}

func (e SnapshotEvent) Validate() error {
	if e.Operation != ValueOperationDelete && e.Operation != ValueOperationUpdate {
		return errors.New("operation must be either delete or update")
	}

	if e.Entitlement.ID == "" {
		return errors.New("entitlementId is required")
	}

	if err := e.Namespace.Validate(); err != nil {
		return err
	}

	if err := e.Subject.Validate(); err != nil {
		return err
	}

	if e.Feature.ID == "" {
		return errors.New("feature ID must be set")
	}

	if e.CalculatedAt == nil {
		return errors.New("calculatedAt is required ")
	}

	switch e.Operation {
	case ValueOperationUpdate:
		if e.Value == nil {
			return errors.New("balance is required for balance update")
		}
	case ValueOperationDelete:
	default:
		return errors.New("operation must be either delete or update")
	}

	return nil
}

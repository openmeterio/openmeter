package schema

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/internal/event/types"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type SubjectKeyAndID struct {
	Key string `json:"key"`
	ID  string `json:"id,omitempty"`
}

func (s SubjectKeyAndID) Validate() error {
	if s.Key == "" {
		return errors.New("key is required")
	}

	return nil
}

type FeatureKeyAndID struct {
	Key string `json:"key"`
	ID  string `json:"id"`
}

func (f FeatureKeyAndID) Validate() error {
	if f.Key == "" {
		return errors.New("key is required")
	}

	if f.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

type BalanceOperationType string

const (
	BalanceOperationUpdate BalanceOperationType = "update"
	BalanceOperationDelete BalanceOperationType = "delete"
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

type EntitlementBalanceSnapshotEvent struct {
	EntitlementID string          `json:"entitlementId"`
	Namespace     string          `json:"namespace"`
	Subject       SubjectKeyAndID `json:"subject"`
	Feature       FeatureKeyAndID `json:"feature"`
	// Operation is delete if the entitlement gets deleted, in that case the balance object is empty
	Operation BalanceOperationType `json:"operation"`

	// CalculatedAt specifies when the balance calculation was performed. It can be used to verify
	// in edge-worker if the store already contains the required item.
	CalculatedAt *time.Time `json:"calculatedAt,omitempty"`

	Balance            *EntitlementValue  `json:"balance,omitempty"`
	CurrentUsagePeriod *recurrence.Period `json:"currentUsagePeriod,omitempty"`
}

var entitlementBalanceSnapshotEventSpec = types.EventTypeSpec{
	Subsystem:   subsystemEntitlement,
	Name:        "snapshot",
	SpecVersion: "1.0",
	Version:     "v1",
	SubjectKind: subjectKindEntitlement,
}

func (e EntitlementBalanceSnapshotEvent) Spec() *types.EventTypeSpec {
	return &entitlementBalanceSnapshotEventSpec
}

func (e EntitlementBalanceSnapshotEvent) Validate() error {
	if e.Operation != BalanceOperationDelete && e.Operation != BalanceOperationUpdate {
		return errors.New("operation must be either delete or update")
	}

	if e.EntitlementID == "" {
		return errors.New("entitlementId is required")
	}

	if e.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := e.Subject.Validate(); err != nil {
		return err
	}

	if err := e.Feature.Validate(); err != nil {
		return err
	}

	if e.Operation == BalanceOperationUpdate {
		if e.CalculatedAt == nil {
			return errors.New("calculatedAt is required for balance update")
		}

		if e.Balance == nil {
			return errors.New("balance is required for balance update")
		}

		if e.CurrentUsagePeriod == nil {
			return errors.New("currentUsagePeriod is required for balance update")
		}
	}

	return nil
}

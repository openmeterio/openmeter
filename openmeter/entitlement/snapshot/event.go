package snapshot

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ValueOperationType string

const (
	ValueOperationReset  ValueOperationType = "reset"
	ValueOperationUpdate ValueOperationType = "update"
	ValueOperationDelete ValueOperationType = "delete"
)

func (o ValueOperationType) Values() []ValueOperationType {
	return []ValueOperationType{
		ValueOperationReset,
		ValueOperationUpdate,
		ValueOperationDelete,
	}
}

func (o ValueOperationType) Validate() error {
	if !slices.Contains(o.Values(), o) {
		return fmt.Errorf("invalid value operation type: %s", o)
	}
	return nil
}

type EntitlementValue struct {
	// Balance Only available for metered entitlements. Metered entitlements are built around a balance calculation where feature usage is deducted from the issued grants. Balance represents the remaining balance of the entitlement, it's value never turns negative.
	Balance *float64 `json:"balance,omitempty"`

	// Config Only available for static entitlements. The JSON parsable config of the entitlement.
	Config *string `json:"config,omitempty"`

	// HasAccess Whether the subject has access to the feature. Shared across all entitlement types.
	HasAccess bool `json:"hasAccess,omitempty"`

	// Overage Only available for metered entitlements. Overage represents the usage that wasn't covered by grants, e.g. if the subject had a total feature usage of 100 in the period but they were only granted 80, there would be 20 overage.
	Overage *float64 `json:"overage,omitempty"`

	// TotalAvailableGrantAmount The summed value of all grant amounts that are active at the time of the query.
	TotalAvailableGrantAmount *float64 `json:"totalAvailableGrantAmount,omitempty"`

	// Usage Only available for metered entitlements. Returns the total feature usage in the current period.
	Usage *float64 `json:"usage,omitempty"`
}

type SnapshotEvent struct {
	Entitlement entitlement.Entitlement `json:"entitlement"`
	Namespace   models.NamespaceID      `json:"namespace"`
	// Deprecated: will be removed when deprecating subjects
	Subject  subject.Subject   `json:"subject"`
	Customer customer.Customer `json:"customer"`
	Feature  feature.Feature   `json:"feature"`
	// Operation is delete if the entitlement gets deleted, in that case the balance object is empty
	Operation ValueOperationType `json:"operation"`

	// CalculatedAt specifies when the balance calculation was performed. It can be used to verify
	// in edge-worker if the store already contains the required item.
	CalculatedAt *time.Time `json:"calculatedAt,omitempty"`

	Value              *EntitlementValue      `json:"value,omitempty"`
	CurrentUsagePeriod *timeutil.ClosedPeriod `json:"currentUsagePeriod,omitempty"`
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
	if e.Customer.ID != "" {
		return metadata.EventMetadata{
			Subject: metadata.ComposeResourcePath(
				e.Namespace.ID,
				metadata.EntityCustomer, e.Customer.ID,
				metadata.EntitySubjectKey, e.Subject.Key,
			),
		}
	}

	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.Entitlement.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityCustomer, e.Customer.ID),
	}
}

func (e SnapshotEvent) Validate() error {
	var errs []error

	if err := e.Operation.Validate(); err != nil {
		errs = append(errs, err)
	}

	if e.Entitlement.ID == "" {
		errs = append(errs, errors.New("entitlementId is required"))
	}

	if err := e.Namespace.Validate(); err != nil {
		errs = append(errs, err)
	}

	// Subject validation is skipped as it's deprecated and may be empty
	// for customers without usage attribution
	// TODO[galexi]: get rid of all references to subject in the codebase

	if e.Feature.ID == "" {
		errs = append(errs, errors.New("feature ID must be set"))
	}

	if err := e.Customer.Validate(); err != nil {
		errs = append(errs, err)
	}

	if e.CalculatedAt == nil {
		errs = append(errs, errors.New("calculatedAt is required "))
	}

	switch e.Operation {
	case ValueOperationUpdate, ValueOperationReset:
		if e.Value == nil {
			errs = append(errs, errors.New("balance is required for balance update/reset"))
		}
	}

	return errors.Join(errs...)
}

// NewSnapshotEvent builds a SnapshotEvent deriving the namespace from the entitlement.
// Though customer and subject properties are in theory present on the entitlement, this constructor uses separate arguments to populate them.
// Subject is deprecated and may be nil for customers without usage attribution.
func NewSnapshotEvent(ent entitlement.Entitlement, subj *subject.Subject, customer customer.Customer, feat feature.Feature, op ValueOperationType, calculatedAt *time.Time, value *EntitlementValue, currentUsagePeriod *timeutil.ClosedPeriod) SnapshotEvent {
	var s subject.Subject
	if subj != nil {
		s = *subj
	}

	return SnapshotEvent{
		Entitlement:        ent,
		Namespace:          models.NamespaceID{ID: ent.Namespace},
		Subject:            s,
		Customer:           customer,
		Feature:            feat,
		Operation:          op,
		CalculatedAt:       calculatedAt,
		Value:              value,
		CurrentUsagePeriod: currentUsagePeriod,
	}
}

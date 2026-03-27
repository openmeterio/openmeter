package persistedstate

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type EntityType string

const (
	EntityTypeLineOrHierarchy EntityType = "line_or_hierarchy"
	EntityTypeCharge          EntityType = "charge"
)

var ErrEntityTypeMismatch = errors.New("entity type mismatch")

type Entity interface {
	IsFlatFee() bool

	GetServicePeriod() timeutil.ClosedPeriod
	GetChildUniqueReferenceID() *string
	GetType() EntityType
	AsLineOrHierarchy() (billing.LineOrHierarchy, error)
	AsCharge() (charges.Charge, error)
}

// Implementations

// Line
var _ Entity = (*LineEntity)(nil)

type LineEntity struct {
	line billing.GenericInvoiceLine
}

func (e LineEntity) GetType() EntityType {
	return EntityTypeLineOrHierarchy
}

func (e LineEntity) AsLineOrHierarchy() (billing.LineOrHierarchy, error) {
	return e.line.AsLineOrHierarchy()
}

// Hierarchy

var _ Entity = (*HierarchyEntity)(nil)

type HierarchyEntity struct {
	hierarchy *billing.SplitLineHierarchy
}

func (e HierarchyEntity) GetType() EntityType {
	return EntityTypeLineOrHierarchy
}

func (e HierarchyEntity) AsLineOrHierarchy() (billing.LineOrHierarchy, error) {
	return billing.NewLineOrHierarchy(e.hierarchy), nil
}

// Charge

var _ Entity = (*ChargeEntity)(nil)

type ChargeEntity struct {
	charge charges.Charge
}

func (e ChargeEntity) GetType() EntityType {
	return EntityTypeCharge
}

func (e ChargeEntity) GetServicePeriod() timeutil.ClosedPeriod {
	return e.charge.GetServicePeriod()
}

func (e ChargeEntity) GetChildUniqueReferenceID() *string {
	return e.charge.GetChildUniqueReferenceID()
}

func (e ChargeEntity) AsLineOrHierarchy() (billing.LineOrHierarchy, error) {
	return billing.LineOrHierarchy{}, ErrEntityTypeMismatch
}

package persistedstate

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Item interface {
	Type() billing.LineOrHierarchyType
	ChildUniqueReferenceID() *string
	ServicePeriod() timeutil.ClosedPeriod
	IsSubscriptionManaged() bool
	HasLastLineAnnotation(annotation string) bool
}

type LineGetter interface {
	GetLine() billing.GenericInvoiceLine
}

type SplitLineHierarchyGetter interface {
	GetSplitLineHierarchy() *billing.SplitLineHierarchy
}

type persistedLine struct {
	line billing.GenericInvoiceLine
}

var (
	_ Item       = persistedLine{}
	_ LineGetter = persistedLine{}
)

// newPersistedLine constructs a persisted line item.
// Kept private so persistedstate controls construction-time validation and Item
// implementations can expose non-erroring accessors.
func newPersistedLine(line billing.GenericInvoiceLine) (persistedLine, error) {
	if line == nil {
		return persistedLine{}, fmt.Errorf("line is nil")
	}

	return persistedLine{line: line}, nil
}

func (i persistedLine) Type() billing.LineOrHierarchyType {
	return billing.LineOrHierarchyTypeLine
}

func (i persistedLine) ChildUniqueReferenceID() *string {
	return i.line.GetChildUniqueReferenceID()
}

func (i persistedLine) ServicePeriod() timeutil.ClosedPeriod {
	return i.line.GetServicePeriod()
}

func (i persistedLine) GetLine() billing.GenericInvoiceLine {
	return i.line
}

func (i persistedLine) IsSubscriptionManaged() bool {
	return i.line.GetManagedBy() == billing.SubscriptionManagedLine
}

func (i persistedLine) HasLastLineAnnotation(annotation string) bool {
	return i.line.GetAnnotations().GetBool(annotation)
}

type persistedSplitLineHierarchy struct {
	hierarchy *billing.SplitLineHierarchy
}

var (
	_ Item                     = persistedSplitLineHierarchy{}
	_ SplitLineHierarchyGetter = persistedSplitLineHierarchy{}
)

// newPersistedSplitLineHierarchy constructs a persisted split line hierarchy item.
// Kept private so persistedstate controls construction-time validation and Item
// implementations can expose non-erroring accessors.
func newPersistedSplitLineHierarchy(hierarchy *billing.SplitLineHierarchy) (persistedSplitLineHierarchy, error) {
	if hierarchy == nil {
		return persistedSplitLineHierarchy{}, fmt.Errorf("split line hierarchy is nil")
	}

	return persistedSplitLineHierarchy{hierarchy: hierarchy}, nil
}

func (i persistedSplitLineHierarchy) Type() billing.LineOrHierarchyType {
	return billing.LineOrHierarchyTypeHierarchy
}

func (i persistedSplitLineHierarchy) ChildUniqueReferenceID() *string {
	return i.hierarchy.Group.UniqueReferenceID
}

func (i persistedSplitLineHierarchy) ServicePeriod() timeutil.ClosedPeriod {
	return i.hierarchy.Group.ServicePeriod.ToClosedPeriod()
}

func (i persistedSplitLineHierarchy) GetSplitLineHierarchy() *billing.SplitLineHierarchy {
	return i.hierarchy
}

func (i persistedSplitLineHierarchy) IsSubscriptionManaged() bool {
	child := i.getLastLineForAnnotations()
	if child == nil {
		return false
	}

	return child.GetManagedBy() == billing.SubscriptionManagedLine
}

func (i persistedSplitLineHierarchy) HasLastLineAnnotation(annotation string) bool {
	child := i.getLastLineForAnnotations()
	if child == nil {
		return false
	}

	return child.GetAnnotations().GetBool(annotation)
}

func (i persistedSplitLineHierarchy) getLastLineForAnnotations() billing.GenericInvoiceLine {
	servicePeriod := i.hierarchy.Group.ServicePeriod
	for _, child := range i.hierarchy.Lines {
		if child.Line.GetServicePeriod().To.Equal(servicePeriod.End) && child.Line.GetDeletedAt() == nil {
			return child.Line
		}
	}

	return nil
}

func NewItemFromLineOrHierarchy(lineOrHierarchy billing.LineOrHierarchy) (Item, error) {
	switch lineOrHierarchy.Type() {
	case billing.LineOrHierarchyTypeLine:
		line, err := lineOrHierarchy.AsGenericLine()
		if err != nil {
			return nil, fmt.Errorf("getting line: %w", err)
		}

		if line == nil {
			return nil, fmt.Errorf("line is nil")
		}

		return newPersistedLine(line)
	case billing.LineOrHierarchyTypeHierarchy:
		hierarchy, err := lineOrHierarchy.AsHierarchy()
		if err != nil {
			return nil, fmt.Errorf("getting hierarchy: %w", err)
		}

		if hierarchy == nil {
			return nil, fmt.Errorf("hierarchy is nil")
		}

		return newPersistedSplitLineHierarchy(hierarchy)
	default:
		return nil, fmt.Errorf("unsupported line or hierarchy type: %s", lineOrHierarchy.Type())
	}
}

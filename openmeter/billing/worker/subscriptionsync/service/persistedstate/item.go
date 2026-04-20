package persistedstate

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ItemType string

const (
	ItemTypeInvoiceLine           ItemType = "invoice.line"
	ItemTypeInvoiceSplitLineGroup ItemType = "invoice.splitLineGroup"
	ItemTypeChargeFlatFee         ItemType = "charge.flatFee"
	ItemTypeChargeUsageBased      ItemType = "charge.usageBased"
)

type Item interface {
	ID() models.NamespacedID
	Type() ItemType
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

type UsageBasedChargeGetter interface {
	GetUsageBasedCharge() usagebased.Charge
}

type FlatFeeChargeGetter interface {
	GetFlatFeeCharge() flatfee.Charge
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

func (i persistedLine) Type() ItemType {
	return ItemTypeInvoiceLine
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

func (i persistedLine) ID() models.NamespacedID {
	lineId := i.line.GetLineID()

	return models.NamespacedID{
		Namespace: lineId.Namespace,
		ID:        lineId.ID,
	}
}

func (i persistedLine) IsSubscriptionManaged() bool {
	return i.line.GetManagedBy() == billing.SubscriptionManagedLine
}

func (i persistedLine) HasLastLineAnnotation(annotation string) bool {
	return i.line.GetAnnotations().GetBool(annotation)
}

// ItemAsLine returns the wrapped line when the persisted item is line-backed.
func ItemAsLine(in Item) (billing.GenericInvoiceLine, error) {
	lineGetter, ok := in.(LineGetter)
	if !ok {
		return nil, fmt.Errorf("persisted item does not implement line getter: %s", getErrorDetails(in))
	}

	return lineGetter.GetLine(), nil
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

func (i persistedSplitLineHierarchy) Type() ItemType {
	return ItemTypeInvoiceSplitLineGroup
}

func (i persistedSplitLineHierarchy) ChildUniqueReferenceID() *string {
	return i.hierarchy.Group.UniqueReferenceID
}

func (i persistedSplitLineHierarchy) ServicePeriod() timeutil.ClosedPeriod {
	return i.hierarchy.Group.ServicePeriod
}

func (i persistedSplitLineHierarchy) GetSplitLineHierarchy() *billing.SplitLineHierarchy {
	return i.hierarchy
}

func (i persistedSplitLineHierarchy) ID() models.NamespacedID {
	return models.NamespacedID{
		Namespace: i.hierarchy.Group.Namespace,
		ID:        i.hierarchy.Group.ID,
	}
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
		if child.Line.GetServicePeriod().To.Equal(servicePeriod.To) && child.Line.GetDeletedAt() == nil {
			return child.Line
		}
	}

	return nil
}

// ItemAsSplitLineHierarchy returns the wrapped hierarchy when the persisted item is hierarchy-backed.
func ItemAsSplitLineHierarchy(in Item) (*billing.SplitLineHierarchy, error) {
	hierarchyGetter, ok := in.(SplitLineHierarchyGetter)
	if !ok {
		return nil, fmt.Errorf("persisted item does not implement split line hierarchy getter: %s", getErrorDetails(in))
	}

	return hierarchyGetter.GetSplitLineHierarchy(), nil
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

type persistedUsageBasedCharge struct {
	charge usagebased.Charge
}

var (
	_ Item                   = persistedUsageBasedCharge{}
	_ UsageBasedChargeGetter = persistedUsageBasedCharge{}
)

// newPersistedUsageBasedCharge constructs a persisted usage-based charge item.
// Kept private so persistedstate controls construction-time validation and Item
// implementations can expose non-erroring accessors.
func newPersistedUsageBasedCharge(charge usagebased.Charge) (persistedUsageBasedCharge, error) {
	if err := charge.Validate(); err != nil {
		return persistedUsageBasedCharge{}, fmt.Errorf("usage based charge is invalid: %w", err)
	}

	return persistedUsageBasedCharge{charge: charge}, nil
}

func (i persistedUsageBasedCharge) ID() models.NamespacedID {
	chargeID := i.charge.GetChargeID()

	return models.NamespacedID{
		Namespace: chargeID.Namespace,
		ID:        chargeID.ID,
	}
}

func (i persistedUsageBasedCharge) Type() ItemType {
	return ItemTypeChargeUsageBased
}

func (i persistedUsageBasedCharge) ChildUniqueReferenceID() *string {
	return i.charge.Intent.UniqueReferenceID
}

func (i persistedUsageBasedCharge) ServicePeriod() timeutil.ClosedPeriod {
	return i.charge.Intent.ServicePeriod
}

func (i persistedUsageBasedCharge) IsSubscriptionManaged() bool {
	return i.charge.Intent.ManagedBy == billing.SubscriptionManagedLine
}

// Charges carry subscription-sync annotations directly on the charge intent, so
// the effective "last line" annotation is always the charge annotation itself.
func (i persistedUsageBasedCharge) HasLastLineAnnotation(annotation string) bool {
	return i.charge.Intent.Annotations.GetBool(annotation)
}

func (i persistedUsageBasedCharge) GetUsageBasedCharge() usagebased.Charge {
	return i.charge
}

// ItemAsUsageBasedCharge returns the wrapped usage-based charge when the persisted item is usage-based.
func ItemAsUsageBasedCharge(in Item) (usagebased.Charge, error) {
	chargeGetter, ok := in.(UsageBasedChargeGetter)
	if !ok {
		return usagebased.Charge{}, fmt.Errorf("persisted item does not implement usage based charge getter: %s", getErrorDetails(in))
	}

	return chargeGetter.GetUsageBasedCharge(), nil
}

type persistedFlatFeeCharge struct {
	charge flatfee.Charge
}

var (
	_ Item                = persistedFlatFeeCharge{}
	_ FlatFeeChargeGetter = persistedFlatFeeCharge{}
)

// newPersistedFlatFeeCharge constructs a persisted flat-fee charge item.
// Kept private so persistedstate controls construction-time validation and Item
// implementations can expose non-erroring accessors.
func newPersistedFlatFeeCharge(charge flatfee.Charge) (persistedFlatFeeCharge, error) {
	if err := charge.Validate(); err != nil {
		return persistedFlatFeeCharge{}, fmt.Errorf("flat fee charge is invalid: %w", err)
	}

	return persistedFlatFeeCharge{charge: charge}, nil
}

func (i persistedFlatFeeCharge) ID() models.NamespacedID {
	chargeID := i.charge.GetChargeID()

	return models.NamespacedID{
		Namespace: chargeID.Namespace,
		ID:        chargeID.ID,
	}
}

func (i persistedFlatFeeCharge) Type() ItemType {
	return ItemTypeChargeFlatFee
}

func (i persistedFlatFeeCharge) ChildUniqueReferenceID() *string {
	return i.charge.Intent.UniqueReferenceID
}

func (i persistedFlatFeeCharge) ServicePeriod() timeutil.ClosedPeriod {
	return i.charge.Intent.ServicePeriod
}

func (i persistedFlatFeeCharge) IsSubscriptionManaged() bool {
	return i.charge.Intent.ManagedBy == billing.SubscriptionManagedLine
}

// Charges carry subscription-sync annotations directly on the charge intent, so
// the effective "last line" annotation is always the charge annotation itself.
func (i persistedFlatFeeCharge) HasLastLineAnnotation(annotation string) bool {
	return i.charge.Intent.Annotations.GetBool(annotation)
}

func (i persistedFlatFeeCharge) GetFlatFeeCharge() flatfee.Charge {
	return i.charge
}

// ItemAsFlatFeeCharge returns the wrapped flat-fee charge when the persisted item is flat-fee-backed.
func ItemAsFlatFeeCharge(in Item) (flatfee.Charge, error) {
	chargeGetter, ok := in.(FlatFeeChargeGetter)
	if !ok {
		return flatfee.Charge{}, fmt.Errorf("persisted item does not implement flat fee charge getter: %s", getErrorDetails(in))
	}

	return chargeGetter.GetFlatFeeCharge(), nil
}

func NewChargeItemFromChargeType(chargeType meta.ChargeType, usageBasedCharge *usagebased.Charge, flatFeeCharge *flatfee.Charge) (Item, error) {
	switch chargeType {
	case meta.ChargeTypeUsageBased:
		if usageBasedCharge == nil {
			return nil, fmt.Errorf("usage based charge is nil")
		}

		return newPersistedUsageBasedCharge(*usageBasedCharge)
	case meta.ChargeTypeFlatFee:
		if flatFeeCharge == nil {
			return nil, fmt.Errorf("flat fee charge is nil")
		}

		return newPersistedFlatFeeCharge(*flatFeeCharge)
	default:
		return nil, fmt.Errorf("unsupported charge type: %s", chargeType)
	}
}

func getErrorDetails(in Item) string {
	return fmt.Sprintf("[id=%s, type=%s]", in.ID(), in.Type())
}

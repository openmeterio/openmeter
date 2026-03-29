package reconciler

import (
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/chargeupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type PatchOperation string

const (
	PatchOperationCreate  PatchOperation = "create"
	PatchOperationDelete  PatchOperation = "delete"
	PatchOperationShrink  PatchOperation = "shrink"
	PatchOperationExtend  PatchOperation = "extend"
	PatchOperationProrate PatchOperation = "prorate"
)

type Patch interface {
	Operation() PatchOperation
	UniqueReferenceID() string
}

type InvoicePatch interface {
	Patch
	GetInvoicePatches() ([]invoiceupdater.Patch, error)
}

type ChargePatch interface {
	Patch
	GetChargePatch() chargeupdater.Patch
}

type InvoicePatchCollection interface {
	Patches() []InvoicePatch
	IsEmpty() bool
}

type ChargePatchCollection interface {
	Patches() []ChargePatch
	IsEmpty() bool
}

type PatchCollection interface {
	AddCreate(target targetstate.StateItem) error
	AddDelete(uniqueID string, existing persistedstate.Item) error
	AddShrink(uniqueID string, existing persistedstate.Item, target targetstate.StateItem) error
	AddExtend(existing persistedstate.Item, target targetstate.StateItem) error
	AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error
}

type patchCollectionRouter struct {
	lineCollection             *lineInvoicePatchCollection
	hierarchyCollection        *lineHierarchyPatchCollection
	flatFeeChargeCollection    *flatFeeChargeCollection
	usageBasedChargeCollection *usageBasedChargeCollection
}

func newPatchCollectionRouter(capacity int, invoices persistedstate.Invoices) (*patchCollectionRouter, error) {
	if invoices == nil {
		return nil, fmt.Errorf("invoices is required")
	}

	lineCollection, err := newLineInvoicePatchCollection(invoices, capacity)
	if err != nil {
		return nil, fmt.Errorf("creating line collection: %w", err)
	}

	return &patchCollectionRouter{
		lineCollection:             lineCollection,
		hierarchyCollection:        newLineHierarchyPatchCollection(capacity),
		flatFeeChargeCollection:    newFlatFeeChargeCollection(capacity),
		usageBasedChargeCollection: newUsageBasedChargeCollection(capacity),
	}, nil
}

func (c patchCollectionRouter) GetCollectionFor(item persistedstate.Item) (PatchCollection, error) {
	switch item.Type() {
	case persistedstate.ItemTypeInvoiceLine:
		return c.lineCollection, nil
	case persistedstate.ItemTypeInvoiceSplitLineGroup:
		return c.hierarchyCollection, nil
	case persistedstate.ItemTypeChargeFlatFee:
		return c.flatFeeChargeCollection, nil
	case persistedstate.ItemTypeChargeUsageBased:
		return c.usageBasedChargeCollection, nil
	default:
		return nil, fmt.Errorf("unsupported persisted item type: %s [id=%s]", item.Type(), item.ID())
	}
}

func (c patchCollectionRouter) ResolveDefaultCollection(target targetstate.StateItem) (PatchCollection, error) {
	if target.Subscription.SettlementMode != productcatalog.CreditOnlySettlementMode {
		return c.lineCollection, nil
	}

	price := target.Spec.RateCard.AsMeta().Price
	if price == nil {
		// This should never happen as we are filtering for !IsBillable() targets in the filterInScopeLinesForInvoiceSync function.
		return nil, fmt.Errorf("price is nil for target[%s]", target.UniqueID)
	}

	switch price.Type() {
	case productcatalog.FlatPriceType:
		return c.flatFeeChargeCollection, nil
	default:
		return c.usageBasedChargeCollection, nil
	}
}

func (c patchCollectionRouter) CollectInvoicePatches() []InvoicePatch {
	allPatches := slices.Concat(c.lineCollection.Patches(), c.hierarchyCollection.Patches())

	filtered := lo.Filter(allPatches, func(patch InvoicePatch, _ int) bool {
		return patch != nil
	})

	return filtered
}

func (c patchCollectionRouter) CollectChargePatches() []ChargePatch {
	allPatches := slices.Concat(c.flatFeeChargeCollection.Patches(), c.usageBasedChargeCollection.Patches())

	filtered := lo.Filter(allPatches, func(patch ChargePatch, _ int) bool {
		return patch != nil
	})

	return filtered
}

package reconciler

import (
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
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

type InvoicePatchCollection interface {
	Patches() []InvoicePatch
	IsEmpty() bool
}

type ChargePatchCollection interface {
	Patches() charges.ApplyPatchesInput
	IsEmpty() bool
}

type PatchCollection interface {
	GetLineEngineType() billing.LineEngineType
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
	creditThenInvoiceEnabled   bool
	creditsEnabled             bool
}

type patchCollectionRouterConfig struct {
	capacity                 int
	invoices                 persistedstate.Invoices
	creditThenInvoiceEnabled bool
	creditsEnabled           bool
}

func (c patchCollectionRouterConfig) Validate() error {
	if c.capacity <= 0 {
		return fmt.Errorf("capacity is required")
	}
	if c.invoices == nil {
		return fmt.Errorf("invoices is required")
	}
	return nil
}

func newPatchCollectionRouter(cfg patchCollectionRouterConfig) (*patchCollectionRouter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	lineCollection, err := newLineInvoicePatchCollection(cfg.invoices, cfg.capacity)
	if err != nil {
		return nil, fmt.Errorf("creating line collection: %w", err)
	}

	return &patchCollectionRouter{
		lineCollection:             lineCollection,
		hierarchyCollection:        newLineHierarchyPatchCollection(cfg.capacity),
		flatFeeChargeCollection:    newFlatFeeChargeCollection(cfg.capacity),
		usageBasedChargeCollection: newUsageBasedChargeCollection(cfg.capacity),
		creditThenInvoiceEnabled:   cfg.creditThenInvoiceEnabled,
		creditsEnabled:             cfg.creditsEnabled,
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
	if !c.creditsEnabled {
		return c.lineCollection, nil
	}

	// If credit then invoice is not enabled, we return the lineCollection.
	if target.Subscription.SettlementMode == productcatalog.CreditThenInvoiceSettlementMode && !c.creditThenInvoiceEnabled {
		return c.lineCollection, nil
	}

	price := target.Spec.RateCard.AsMeta().Price
	if price == nil {
		// This should never happen as we are filtering for !IsBillable() targets in the filterInScopeLines function.
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

func (c patchCollectionRouter) CollectChargePatches() (charges.ApplyPatchesInput, error) {
	return charges.ConcatenateApplyPatchesInputs(
		c.flatFeeChargeCollection.Patches(),
		c.usageBasedChargeCollection.Patches(),
	)
}

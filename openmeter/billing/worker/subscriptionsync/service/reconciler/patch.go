package reconciler

import (
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
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

type PatchCollection interface {
	AddCreate(target targetstate.StateItem) error
	AddDelete(uniqueID string, existing persistedstate.Item) error
	AddShrink(uniqueID string, existing persistedstate.Item, target targetstate.StateItem) error
	AddExtend(existing persistedstate.Item, target targetstate.StateItem) error
	AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error
}

type patchCollectionRouter struct {
	lineCollection      *lineInvoicePatchCollection
	hierarchyCollection *lineHierarchyPatchCollection
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
		lineCollection:      lineCollection,
		hierarchyCollection: newLineHierarchyPatchCollection(capacity),
	}, nil
}

func (c patchCollectionRouter) GetCollectionFor(item persistedstate.Item) (PatchCollection, error) {
	switch item.Type() {
	case billing.LineOrHierarchyTypeLine:
		return c.lineCollection, nil
	case billing.LineOrHierarchyTypeHierarchy:
		return c.hierarchyCollection, nil
	default:
		return nil, fmt.Errorf("unsupported persisted item type: %s [id=%s]", item.Type(), item.ID())
	}
}

func (c patchCollectionRouter) ResolveDefaultCollection() PatchCollection {
	// TODO: Once we have charges wired in we need a helper function to determine the default routing for new lines depending on the
	// settlement type set on the subscription and feature flags in the config of subscription sync.
	return c.lineCollection
}

func (c patchCollectionRouter) CollectInvoicePatches() []InvoicePatch {
	allPatches := slices.Concat(c.lineCollection.Patches(), c.hierarchyCollection.Patches())

	filtered := lo.Filter(allPatches, func(patch InvoicePatch, _ int) bool {
		return patch != nil
	})

	return filtered
}

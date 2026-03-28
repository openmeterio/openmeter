package reconciler

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

type GetInvoicePatchesInput struct {
	Subscription subscription.Subscription
	Currency     currencyx.Calculator
	Invoices     persistedstate.Invoices
}

type Patch interface {
	Operation() PatchOperation
	UniqueReferenceID() string
}

type InvoicePatch interface {
	Patch
	GetInvoicePatches(input GetInvoicePatchesInput) ([]invoiceupdater.Patch, error)
}

type InvoicePatchCollection interface {
	Patches() []InvoicePatch
	IsEmpty() bool
}

type PatchCollection interface {
	AddCreate(target targetstate.StateItem)
	AddDelete(uniqueID string, existing persistedstate.Item) error
	AddShrink(uniqueID string, existing persistedstate.Item, target targetstate.StateItem) error
	AddExtend(existing persistedstate.Item, target targetstate.StateItem) error
	AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error
}

type invoicePatchCollection struct {
	patches []InvoicePatch
}

func newInvoicePatchCollection(capacity int) *invoicePatchCollection {
	return &invoicePatchCollection{
		patches: make([]InvoicePatch, 0, capacity),
	}
}

func (c *invoicePatchCollection) AddCreate(target targetstate.StateItem) {
	c.patches = append(c.patches, CreatePatch{
		Target: target,
	})
}

func (c *invoicePatchCollection) AddDelete(uniqueID string, existing persistedstate.Item) error {
	existingLineOrHierarchy, err := persistedItemAsLineOrHierarchy(existing)
	if err != nil {
		return fmt.Errorf("converting existing item to line or hierarchy: %w", err)
	}

	c.patches = append(c.patches, DeletePatch{
		UniqueID: uniqueID,
		Existing: existingLineOrHierarchy,
	})

	return nil
}

func (c *invoicePatchCollection) AddShrink(uniqueID string, existing persistedstate.Item, target targetstate.StateItem) error {
	existingLineOrHierarchy, err := persistedItemAsLineOrHierarchy(existing)
	if err != nil {
		return fmt.Errorf("converting existing item to line or hierarchy: %w", err)
	}

	c.patches = append(c.patches, ShrinkUsageBasedPatch{
		UniqueID: uniqueID,
		Existing: existingLineOrHierarchy,
		Target:   target,
	})

	return nil
}

func (c *invoicePatchCollection) AddExtend(existing persistedstate.Item, target targetstate.StateItem) error {
	existingLineOrHierarchy, err := persistedItemAsLineOrHierarchy(existing)
	if err != nil {
		return fmt.Errorf("converting existing item to line or hierarchy: %w", err)
	}

	c.patches = append(c.patches, ExtendUsageBasedPatch{
		Existing: existingLineOrHierarchy,
		Target:   target,
	})

	return nil
}

func (c *invoicePatchCollection) AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error {
	existingLineOrHierarchy, err := persistedItemAsLineOrHierarchy(existing)
	if err != nil {
		return fmt.Errorf("converting existing item to line or hierarchy: %w", err)
	}

	c.patches = append(c.patches, ProratePatch{
		Existing:       existingLineOrHierarchy,
		Target:         target,
		OriginalPeriod: originalPeriod,
		TargetPeriod:   targetPeriod,
		OriginalAmount: originalAmount,
		TargetAmount:   targetAmount,
	})

	return nil
}

func (c *invoicePatchCollection) Patches() []InvoicePatch {
	return c.patches
}

func (c *invoicePatchCollection) IsEmpty() bool {
	return len(c.patches) == 0
}

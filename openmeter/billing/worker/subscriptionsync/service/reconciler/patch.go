package reconciler

import (
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/featuregate"
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

// ChargeReferencePatches is kept separate from ordinary charge patches so the same
// charge can first repair its physical subscription ownership and then receive one
// compatible lifecycle patch. The map key enforces one reference repair per charge.
type ChargeReferencePatches map[chargesmeta.ChargeID]chargesmeta.PatchUpdateSubscriptionReference

func (p ChargeReferencePatches) IsEmpty() bool {
	return len(p) == 0
}

func (p ChargeReferencePatches) add(chargeID chargesmeta.ChargeID, patch chargesmeta.PatchUpdateSubscriptionReference) error {
	if p == nil {
		return fmt.Errorf("charge reference patches are required")
	}

	if err := chargeID.Validate(); err != nil {
		return fmt.Errorf("invalid charge ID: %w", err)
	}
	if err := patch.Validate(); err != nil {
		return fmt.Errorf("invalid subscription reference patch: %w", err)
	}

	if _, exists := p[chargeID]; exists {
		return fmt.Errorf("subscription reference patch for charge ID %s already exists", chargeID.ID)
	}

	p[chargeID] = patch

	return nil
}

func (p ChargeReferencePatches) asApplyPatchesInput(customerID customer.CustomerID) (charges.ApplyPatchesInput, error) {
	patchesByChargeID := make(map[string]charges.Patch, len(p))
	for chargeID, patch := range p {
		if err := chargeID.Validate(); err != nil {
			return charges.ApplyPatchesInput{}, fmt.Errorf("invalid charge ID: %w", err)
		}
		if chargeID.Namespace != customerID.Namespace {
			return charges.ApplyPatchesInput{}, fmt.Errorf("charge[%s] namespace does not match customer namespace", chargeID.ID)
		}
		if err := patch.Validate(); err != nil {
			return charges.ApplyPatchesInput{}, fmt.Errorf("invalid subscription reference patch for charge[%s]: %w", chargeID.ID, err)
		}

		patchesByChargeID[chargeID.ID] = patch
	}

	return charges.ApplyPatchesInput{
		CustomerID:        customerID,
		PatchesByChargeID: patchesByChargeID,
	}, nil
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
	featureGate                *featuregate.FeatureGateChecker
}

type patchCollectionRouterConfig struct {
	capacity                 int
	invoices                 persistedstate.Invoices
	creditThenInvoiceEnabled bool
	creditsEnabled           bool
	featureGate              *featuregate.FeatureGateChecker
}

func (c patchCollectionRouterConfig) Validate() error {
	if c.capacity <= 0 {
		return fmt.Errorf("capacity is required")
	}

	if c.invoices == nil {
		return fmt.Errorf("invoices is required")
	}

	if err := c.featureGate.Validate(); err != nil {
		return err
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
		featureGate:                cfg.featureGate,
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

func (c patchCollectionRouter) isCreditsEnabled(ns string) (bool, error) {
	if !c.creditsEnabled {
		return false, nil
	}
	return c.featureGate.Enabled(ns, c.featureGate.Flags.Credits())
}

func (c patchCollectionRouter) ResolveDefaultCollection(target targetstate.StateItem) (PatchCollection, error) {
	enabled, err := c.isCreditsEnabled(target.SubscriptionItem.NamespacedID.Namespace)
	if err != nil {
		return nil, err
	}

	if !enabled {
		if target.Currency.IsCustom() {
			return nil, fmt.Errorf("custom currency subscription items require the charges service [currency=%s]", target.Currency.GetCode())
		}

		return c.lineCollection, nil
	}

	// If credit then invoice is not enabled, we return the lineCollection.
	if target.Subscription.SettlementMode == productcatalog.CreditThenInvoiceSettlementMode && !c.creditThenInvoiceEnabled {
		if target.Currency.IsCustom() {
			return nil, fmt.Errorf("custom currency credit-then-invoice subscription items require charge-based credit-then-invoice billing [currency=%s]", target.Currency.GetCode())
		}

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

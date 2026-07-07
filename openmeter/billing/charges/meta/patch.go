package meta

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PatchType string

const (
	PatchTypeExtend         PatchType = "extend"
	PatchTypeShrink         PatchType = "shrink"
	PatchTypeDelete         PatchType = "delete"
	PatchTypeLineManualEdit PatchType = "line_manual_edit"
)

type ChangeTarget string

const (
	ChangeTargetBase     ChangeTarget = "base"
	ChangeTargetOverride ChangeTarget = "override"
)

func (t ChangeTarget) Values() []ChangeTarget {
	return []ChangeTarget{
		ChangeTargetBase,
		ChangeTargetOverride,
	}
}

func (t ChangeTarget) Validate() error {
	if !slices.Contains(t.Values(), t) {
		return models.NewGenericValidationError(fmt.Errorf("invalid change target: %s", t))
	}

	return nil
}

type LayeredIntentReader interface {
	GetBaseManagedBy() billing.InvoiceLineManagedBy
	HasOverrideLayer() bool
}

func apiPatchTargetLayer(intent LayeredIntentReader) (ChangeTarget, error) {
	if intent == nil {
		return "", errors.New("intent is required")
	}

	if intent.HasOverrideLayer() || intent.GetBaseManagedBy() != billing.ManuallyManagedLine {
		return ChangeTargetOverride, nil
	}

	return ChangeTargetBase, nil
}

type Patch interface {
	models.Validator

	Op() PatchType
	Trigger() stateless.Trigger
}

type TriggerPatchResult[T any] struct {
	Charge         *T
	InvoicePatches invoiceupdater.Patches
}

// PatchAction adapts a generic Patch action to a concrete patch action when
// statelessx.AllOfWithParameters requires strict typing for composed actions.
func PatchAction[T Patch](fn func(context.Context, Patch) error) func(context.Context, T) error {
	return func(ctx context.Context, patch T) error {
		return fn(ctx, patch)
	}
}

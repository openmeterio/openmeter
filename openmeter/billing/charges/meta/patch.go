package meta

import (
	"fmt"
	"slices"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PatchType string

const (
	PatchTypeExtend     PatchType = "extend"
	PatchTypeShrink     PatchType = "shrink"
	PatchTypeDelete     PatchType = "delete"
	PatchTypeManualEdit PatchType = "manual_edit"
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

type Patch interface {
	models.Validator

	Op() PatchType
	GetTarget() ChangeTarget
	Trigger() stateless.Trigger
}

type TriggerPatchResult[T any] struct {
	Charge         *T
	InvoicePatches []invoiceupdater.Patch
}

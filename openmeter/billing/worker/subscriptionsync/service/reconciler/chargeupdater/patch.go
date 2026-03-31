package chargeupdater

import (
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

type PatchOperation string

const (
	PatchOpCreate PatchOperation = "create"
)

type PatchCreate struct {
	Intent charges.ChargeIntent
}

type Patch struct {
	op PatchOperation

	createPatch PatchCreate
}

func (p Patch) Op() PatchOperation {
	return p.op
}

func (p Patch) AsCreatePatch() (PatchCreate, error) {
	if p.op != PatchOpCreate {
		return PatchCreate{}, fmt.Errorf("expected create patch, got %s", p.op)
	}

	return p.createPatch, nil
}

func NewCreatePatch(intent charges.ChargeIntent) Patch {
	return Patch{
		op: PatchOpCreate,
		createPatch: PatchCreate{
			Intent: intent,
		},
	}
}

func (p Patch) Log(logger *slog.Logger) {
	switch p.op {
	case PatchOpCreate:
		logger.Info("create charge patch", "charge_type", p.createPatch.Intent.Type())
	default:
		logger.Info("unknown patch operation", "operation", p.op)
	}
}

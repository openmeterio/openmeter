package invoiceupdater

import (
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type PatchOperation string

const (
	PatchOpLineCreate PatchOperation = "line_create"
	PatchOpLineDelete PatchOperation = "line_delete"
	PatchOpLineUpdate PatchOperation = "line_update"
)

type PatchLineCreate struct {
	Line billing.GatheringLine
}

type PatchLineDelete struct {
	Line      billing.LineID
	InvoiceID string
}

type PatchLineUpdate struct {
	TargetState billing.GenericInvoiceLine
}

type Patch struct {
	op PatchOperation

	createLinePatch PatchLineCreate
	deleteLinePatch PatchLineDelete
	updateLinePatch PatchLineUpdate
}

func (p Patch) Op() PatchOperation {
	return p.op
}

func (p Patch) AsCreateLinePatch() (PatchLineCreate, error) {
	if p.op != PatchOpLineCreate {
		return PatchLineCreate{}, fmt.Errorf("expected create line patch, got %s", p.op)
	}

	return p.createLinePatch, nil
}

func (p Patch) AsDeleteLinePatch() (PatchLineDelete, error) {
	if p.op != PatchOpLineDelete {
		return PatchLineDelete{}, fmt.Errorf("expected delete line patch, got %s", p.op)
	}

	return p.deleteLinePatch, nil
}

func (p Patch) AsUpdateLinePatch() (PatchLineUpdate, error) {
	if p.op != PatchOpLineUpdate {
		return PatchLineUpdate{}, fmt.Errorf("expected update line patch, got %s", p.op)
	}

	return p.updateLinePatch, nil
}

func NewDeleteLinePatch(lineID billing.LineID, invoiceID string) Patch {
	return Patch{
		op: PatchOpLineDelete,
		deleteLinePatch: PatchLineDelete{
			Line:      lineID,
			InvoiceID: invoiceID,
		},
	}
}

func NewUpdateLinePatch(line billing.GenericInvoiceLine) Patch {
	return Patch{
		op: PatchOpLineUpdate,
		updateLinePatch: PatchLineUpdate{
			TargetState: line,
		},
	}
}

func NewCreateLinePatch(line billing.GatheringLine) Patch {
	return Patch{
		op: PatchOpLineCreate,
		createLinePatch: PatchLineCreate{
			Line: line,
		},
	}
}

func (p Patch) Log(logger *slog.Logger) {
	switch p.op {
	case PatchOpLineCreate:
		logger.Info("create line patch", "line_id", p.createLinePatch.Line.GetLineID().ID, "new_service_period_from", p.createLinePatch.Line.GetServicePeriod().From, "new_service_period_to", p.createLinePatch.Line.GetServicePeriod().To, "unique_reference_id", p.createLinePatch.Line.GetChildUniqueReferenceID())
	case PatchOpLineDelete:
		logger.Info("delete line patch", "line_id", p.deleteLinePatch.Line, "invoice_id", p.deleteLinePatch.InvoiceID)
	case PatchOpLineUpdate:
		logger.Info("update line patch", "line_id", p.updateLinePatch.TargetState.GetLineID().ID, "invoice_id", p.updateLinePatch.TargetState.GetInvoiceID(), "new_service_period_from", p.updateLinePatch.TargetState.GetServicePeriod().From, "new_service_period_to", p.updateLinePatch.TargetState.GetServicePeriod().To, "unique_reference_id", p.updateLinePatch.TargetState.GetChildUniqueReferenceID())
	default:
		logger.Info("unknown patch operation", "operation", p.op)
	}
}

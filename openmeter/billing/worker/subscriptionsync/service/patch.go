package service

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/pkg/models"
)

type patchOperation string

const (
	patchOpLineCreate                      patchOperation = "line_create"
	patchOpLineDelete                      patchOperation = "line_delete"
	patchOpLineUpdate                      patchOperation = "line_update"
	patchOpSplitLineGroupDelete            patchOperation = "split_line_group_delete"
	patchOpSplitLineGroupUpdate            patchOperation = "split_line_group_update"
	patchOpUpsertChargeAndAssociateLines   patchOperation = "upsert_charge_and_associate_lines"
	patchOpDeleteChargeByUniqueReferenceID patchOperation = "delete_charge_by_unique_reference_id"
)

type linePatchLineCreate struct {
	Line billing.GatheringLine
}

type linePatchLineDelete struct {
	Line      billing.LineID
	InvoiceID string
}

type linePatchLineUpdate struct {
	TargetState billing.GenericInvoiceLine
}

type linePatchSplitLineGroupDelete struct {
	Group models.NamespacedID
}

type linePatchSplitLineGroupUpdate struct {
	TargetState billing.SplitLineGroupUpdate
}

type upsertChargeAndAssociateLinesPatch struct {
	Charge                      charges.Charge
	LinesIDsToAssociate         []billing.LineID
	SplitLineGroupIDToAssociate *models.NamespacedID
}

type deleteChargeByUniqueReferenceIDPatch struct {
	UniqueReferenceID string
}

type linePatch struct {
	op patchOperation

	createLinePatch linePatchLineCreate
	deleteLinePatch linePatchLineDelete
	updateLinePatch linePatchLineUpdate

	deleteSplitLineGroupPatch linePatchSplitLineGroupDelete
	updateSplitLineGroupPatch linePatchSplitLineGroupUpdate

	// Charges
	upsertChargeAndAssociateLinesPatch   upsertChargeAndAssociateLinesPatch
	deleteChargeByUniqueReferenceIDPatch deleteChargeByUniqueReferenceIDPatch
}

func (p linePatch) Op() patchOperation {
	return p.op
}

func (p linePatch) AsCreateLinePatch() (linePatchLineCreate, error) {
	if p.op != patchOpLineCreate {
		return linePatchLineCreate{}, fmt.Errorf("expected create line patch, got %s", p.op)
	}

	return p.createLinePatch, nil
}

func (p linePatch) AsDeleteLinePatch() (linePatchLineDelete, error) {
	if p.op != patchOpLineDelete {
		return linePatchLineDelete{}, fmt.Errorf("expected delete line patch, got %s", p.op)
	}

	return p.deleteLinePatch, nil
}

func (p linePatch) AsUpdateLinePatch() (linePatchLineUpdate, error) {
	if p.op != patchOpLineUpdate {
		return linePatchLineUpdate{}, fmt.Errorf("expected update line patch, got %s", p.op)
	}

	return p.updateLinePatch, nil
}

func (p linePatch) AsDeleteSplitLineGroupPatch() (linePatchSplitLineGroupDelete, error) {
	if p.op != patchOpSplitLineGroupDelete {
		return linePatchSplitLineGroupDelete{}, fmt.Errorf("expected delete split line group patch, got %s", p.op)
	}

	return p.deleteSplitLineGroupPatch, nil
}

func (p linePatch) AsUpdateSplitLineGroupPatch() (linePatchSplitLineGroupUpdate, error) {
	if p.op != patchOpSplitLineGroupUpdate {
		return linePatchSplitLineGroupUpdate{}, fmt.Errorf("expected update split line group patch, got %s", p.op)
	}

	return p.updateSplitLineGroupPatch, nil
}

func (p linePatch) AsUpsertChargeAndAssociateLinesPatch() (upsertChargeAndAssociateLinesPatch, error) {
	if p.op != patchOpUpsertChargeAndAssociateLines {
		return upsertChargeAndAssociateLinesPatch{}, fmt.Errorf("expected upsert charge and associate lines patch, got %s", p.op)
	}

	return p.upsertChargeAndAssociateLinesPatch, nil
}

func (p linePatch) AsDeleteChargeByUniqueReferenceIDPatch() (deleteChargeByUniqueReferenceIDPatch, error) {
	if p.op != patchOpDeleteChargeByUniqueReferenceID {
		return deleteChargeByUniqueReferenceIDPatch{}, fmt.Errorf("expected delete charge by unique reference ID patch, got %s", p.op)
	}

	return p.deleteChargeByUniqueReferenceIDPatch, nil
}

func newDeleteLinePatch(lineID billing.LineID, invoiceID string) linePatch {
	return linePatch{
		op: patchOpLineDelete,
		deleteLinePatch: linePatchLineDelete{
			Line:      lineID,
			InvoiceID: invoiceID,
		},
	}
}

func newUpdateLinePatch(line billing.GenericInvoiceLine) linePatch {
	return linePatch{
		op: patchOpLineUpdate,
		updateLinePatch: linePatchLineUpdate{
			TargetState: line,
		},
	}
}

func newDeleteSplitLineGroupPatch(groupID models.NamespacedID) linePatch {
	return linePatch{
		op: patchOpSplitLineGroupDelete,
		deleteSplitLineGroupPatch: linePatchSplitLineGroupDelete{
			Group: groupID,
		},
	}
}

func newUpdateSplitLineGroupPatch(group billing.SplitLineGroupUpdate) linePatch {
	return linePatch{
		op: patchOpSplitLineGroupUpdate,
		updateSplitLineGroupPatch: linePatchSplitLineGroupUpdate{
			TargetState: group,
		},
	}
}

func newCreateLinePatch(line billing.GatheringLine) linePatch {
	return linePatch{
		op: patchOpLineCreate,
		createLinePatch: linePatchLineCreate{
			Line: line,
		},
	}
}

func newUpsertChargeAndAssociateLinesPatch(charge charges.Charge, linesIDsToAssociate ...billing.LineID) linePatch {
	return linePatch{
		op: patchOpUpsertChargeAndAssociateLines,
		upsertChargeAndAssociateLinesPatch: upsertChargeAndAssociateLinesPatch{
			Charge:              charge,
			LinesIDsToAssociate: linesIDsToAssociate,
		},
	}
}

func newUpsertChargeAndAssociateLinesPatchWithSplitLineGroup(charge charges.Charge, linesIDsToAssociate []billing.LineID, splitLineGroupID models.NamespacedID) linePatch {
	return linePatch{
		op: patchOpUpsertChargeAndAssociateLines,
		upsertChargeAndAssociateLinesPatch: upsertChargeAndAssociateLinesPatch{
			Charge:                      charge,
			LinesIDsToAssociate:         linesIDsToAssociate,
			SplitLineGroupIDToAssociate: &splitLineGroupID,
		},
	}
}

func newDeleteChargeByUniqueReferenceIDPatch(uniqueReferenceID string) linePatch {
	return linePatch{
		op: patchOpDeleteChargeByUniqueReferenceID,
		deleteChargeByUniqueReferenceIDPatch: deleteChargeByUniqueReferenceIDPatch{
			UniqueReferenceID: uniqueReferenceID,
		},
	}
}

func (s *Service) getDeletePatchesForLine(lineOrHierarchy billing.LineOrHierarchy) ([]linePatch, error) {
	switch lineOrHierarchy.Type() {
	case billing.LineOrHierarchyTypeLine:
		line, err := lineOrHierarchy.AsGenericLine()
		if err != nil {
			return nil, fmt.Errorf("getting line: %w", err)
		}

		chargePatch := []linePatch{}
		if uniqueReferenceID := line.GetChildUniqueReferenceID(); uniqueReferenceID != nil {
			chargePatch = append(chargePatch, newDeleteChargeByUniqueReferenceIDPatch(*uniqueReferenceID))
		}

		// Ignored lines do not take part in syncing so we skip them
		if line.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
			return chargePatch, nil
		}

		return slices.Concat(
			chargePatch,
			[]linePatch{newDeleteLinePatch(line.GetLineID(), line.GetInvoiceID())},
		), nil
	case billing.LineOrHierarchyTypeHierarchy:
		group, err := lineOrHierarchy.AsHierarchy()
		if err != nil {
			return nil, fmt.Errorf("getting split line hierarchy: %w", err)
		}

		out := make([]linePatch, 0, 1+len(group.Lines))

		if group.Group.DeletedAt == nil {
			out = append(out, newDeleteSplitLineGroupPatch(models.NamespacedID{
				Namespace: group.Group.Namespace,
				ID:        group.Group.ID,
			}))
		}

		for _, line := range group.Lines {
			if line.Line.GetDeletedAt() != nil {
				continue
			}

			out = append(out, newDeleteLinePatch(line.Line.GetLineID(), line.Invoice.GetID()))
		}

		if group.Group.UniqueReferenceID != nil {
			out = append(out, newDeleteChargeByUniqueReferenceIDPatch(*group.Group.UniqueReferenceID))
		}

		return out, nil
	}

	return nil, fmt.Errorf("unsupported line or hierarchy type: %s", lineOrHierarchy.Type())
}

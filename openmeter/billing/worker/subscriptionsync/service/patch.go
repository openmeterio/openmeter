package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

type patchOperation string

const (
	patchOpLineCreate           patchOperation = "line_create"
	patchOpLineDelete           patchOperation = "line_delete"
	patchOpLineUpdate           patchOperation = "line_update"
	patchOpSplitLineGroupDelete patchOperation = "split_line_group_delete"
	patchOpSplitLineGroupUpdate patchOperation = "split_line_group_update"
)

type linePatchLineCreate struct {
	Line billing.Line
}

type linePatchLineDelete struct {
	Line      billing.LineID
	InvoiceID string
}

type linePatchLineUpdate struct {
	TargetState *billing.Line
}

type linePatchSplitLineGroupDelete struct {
	Group models.NamespacedID
}

type linePatchSplitLineGroupUpdate struct {
	TargetState billing.SplitLineGroupUpdate
}

type linePatch struct {
	op patchOperation

	createLinePatch linePatchLineCreate
	deleteLinePatch linePatchLineDelete
	updateLinePatch linePatchLineUpdate

	deleteSplitLineGroupPatch linePatchSplitLineGroupDelete
	updateSplitLineGroupPatch linePatchSplitLineGroupUpdate
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

func newDeleteLinePatch(lineID billing.LineID, invoiceID string) linePatch {
	return linePatch{
		op: patchOpLineDelete,
		deleteLinePatch: linePatchLineDelete{
			Line:      lineID,
			InvoiceID: invoiceID,
		},
	}
}

func newUpdateLinePatch(line *billing.Line) linePatch {
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

func newCreateLinePatch(line billing.Line) linePatch {
	return linePatch{
		op: patchOpLineCreate,
		createLinePatch: linePatchLineCreate{
			Line: line,
		},
	}
}

func (s *Service) getDeletePatchesForLine(lineOrHierarchy billing.LineOrHierarchy) ([]linePatch, error) {
	switch lineOrHierarchy.Type() {
	case billing.LineOrHierarchyTypeLine:
		line, err := lineOrHierarchy.AsLine()
		if err != nil {
			return nil, fmt.Errorf("getting line: %w", err)
		}

		// Ignored lines do not take part in syncing so we skip them
		if line.Annotations.GetBool(billing.AnnotationSubscriptionSyncIgnore) {
			return nil, nil
		}

		return []linePatch{
			newDeleteLinePatch(line.LineID(), line.InvoiceID),
		}, nil
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
			if line.Line.DeletedAt != nil {
				continue
			}

			out = append(out, newDeleteLinePatch(line.Line.LineID(), line.Line.InvoiceID))
		}

		return out, nil
	}

	return nil, fmt.Errorf("unsupported line or hierarchy type: %s", lineOrHierarchy.Type())
}

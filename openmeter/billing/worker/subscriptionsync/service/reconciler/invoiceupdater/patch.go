package invoiceupdater

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PatchOperation string

const (
	PatchOpLineCreate           PatchOperation = "line_create"
	PatchOpLineDelete           PatchOperation = "line_delete"
	PatchOpLineUpdate           PatchOperation = "line_update"
	PatchOpSplitLineGroupDelete PatchOperation = "split_line_group_delete"
	PatchOpSplitLineGroupUpdate PatchOperation = "split_line_group_update"
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

type PatchSplitLineGroupDelete struct {
	Group models.NamespacedID
}

type PatchSplitLineGroupUpdate struct {
	TargetState billing.SplitLineGroupUpdate
}

type Patch struct {
	op PatchOperation

	createLinePatch PatchLineCreate
	deleteLinePatch PatchLineDelete
	updateLinePatch PatchLineUpdate

	deleteSplitLineGroupPatch PatchSplitLineGroupDelete
	updateSplitLineGroupPatch PatchSplitLineGroupUpdate
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

func (p Patch) AsDeleteSplitLineGroupPatch() (PatchSplitLineGroupDelete, error) {
	if p.op != PatchOpSplitLineGroupDelete {
		return PatchSplitLineGroupDelete{}, fmt.Errorf("expected delete split line group patch, got %s", p.op)
	}

	return p.deleteSplitLineGroupPatch, nil
}

func (p Patch) AsUpdateSplitLineGroupPatch() (PatchSplitLineGroupUpdate, error) {
	if p.op != PatchOpSplitLineGroupUpdate {
		return PatchSplitLineGroupUpdate{}, fmt.Errorf("expected update split line group patch, got %s", p.op)
	}

	return p.updateSplitLineGroupPatch, nil
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

func NewDeleteSplitLineGroupPatch(groupID models.NamespacedID) Patch {
	return Patch{
		op: PatchOpSplitLineGroupDelete,
		deleteSplitLineGroupPatch: PatchSplitLineGroupDelete{
			Group: groupID,
		},
	}
}

func NewUpdateSplitLineGroupPatch(group billing.SplitLineGroupUpdate) Patch {
	return Patch{
		op: PatchOpSplitLineGroupUpdate,
		updateSplitLineGroupPatch: PatchSplitLineGroupUpdate{
			TargetState: group,
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

func GetDeletePatchesForLine(lineOrHierarchy billing.LineOrHierarchy) ([]Patch, error) {
	switch lineOrHierarchy.Type() {
	case billing.LineOrHierarchyTypeLine:
		line, err := lineOrHierarchy.AsGenericLine()
		if err != nil {
			return nil, fmt.Errorf("getting line: %w", err)
		}

		if line.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
			return nil, nil
		}

		if line.GetDeletedAt() != nil {
			return nil, nil
		}

		return []Patch{
			NewDeleteLinePatch(line.GetLineID(), line.GetInvoiceID()),
		}, nil
	case billing.LineOrHierarchyTypeHierarchy:
		group, err := lineOrHierarchy.AsHierarchy()
		if err != nil {
			return nil, fmt.Errorf("getting split line hierarchy: %w", err)
		}

		out := make([]Patch, 0, 1+len(group.Lines))

		// Skip the group if any of the lines are ignored
		for _, line := range group.Lines {
			if line.Line.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
				return nil, nil
			}
		}

		if group.Group.DeletedAt == nil {
			out = append(out, NewDeleteSplitLineGroupPatch(models.NamespacedID{
				Namespace: group.Group.Namespace,
				ID:        group.Group.ID,
			}))
		}

		for _, line := range group.Lines {
			if line.Line.GetDeletedAt() != nil {
				continue
			}

			out = append(out, NewDeleteLinePatch(line.Line.GetLineID(), line.Invoice.GetID()))
		}

		return out, nil
	}

	return nil, fmt.Errorf("unsupported line or hierarchy type: %s", lineOrHierarchy.Type())
}

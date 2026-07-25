package invoiceupdater

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type Patches []Patch

// BisectByInvoiceID splits the patches into two groups: one for the invoice ID before the patch and one for the invoice ID after the patch.
//
// First return value is the patches with the invoice ID, the second return value is the patches without the invoice ID.
// Corner cases:
// - Any gathering invoice line patch will be in the `rest` group.
// - Any create line patch will be in the `rest` group.
func (p Patches) BisectByStandardInvoiceID(invoiceID string) (Patches, Patches, error) {
	invoicePatches := make(Patches, 0, len(p))
	rest := make(Patches, 0, len(p))

	for _, patch := range p {
		switch patch.Op() {
		case PatchOpLineDelete:
			val, err := patch.AsDeleteLinePatch()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert patch to delete line patch: %w", err)
			}
			if val.InvoiceID == invoiceID {
				invoicePatches = append(invoicePatches, patch)
			} else {
				rest = append(rest, patch)
			}
		case PatchOpLineUpdate:
			val, err := patch.AsUpdateLinePatch()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert patch to update line patch: %w", err)
			}
			if val.TargetState.GetInvoiceID() == invoiceID {
				invoicePatches = append(invoicePatches, patch)
			} else {
				rest = append(rest, patch)
			}
		default:
			rest = append(rest, patch)
		}
	}

	return invoicePatches, rest, nil
}

func (p Patches) RequireSingularStandardInvoiceLineDeletePatch() (PatchLineDelete, error) {
	patch, err := p.requireSingularPatch("standard invoice line delete")
	if err != nil {
		return PatchLineDelete{}, err
	}

	return patch.AsDeleteLinePatch()
}

func (p Patches) RequireSingularLineUpdatePatchForTarget(line billing.GenericInvoiceLineReader) (PatchLineUpdate, error) {
	patch, err := p.requireSingularPatch("line update")
	if err != nil {
		return PatchLineUpdate{}, err
	}

	updatePatch, err := patch.AsUpdateLinePatch()
	if err != nil {
		return PatchLineUpdate{}, err
	}

	if err := updatePatch.RequireTarget(line); err != nil {
		return PatchLineUpdate{}, err
	}

	return updatePatch, nil
}

func (p Patches) RequireSingularGatheringLinePatchForCharge(chargeID string) (Patch, error) {
	patch, err := p.requireSingularPatch("gathering line by charge")
	if err != nil {
		return Patch{}, err
	}

	switch patch.Op() {
	case PatchOpUpsertGatheringLineByChargeID:
		upsertPatch, err := patch.AsUpsertGatheringLineByChargeIDPatch()
		if err != nil {
			return Patch{}, err
		}

		if err := upsertPatch.RequireCharge(chargeID); err != nil {
			return Patch{}, err
		}

		return patch, nil
	case PatchOpDeleteGatheringLineByChargeID:
		deletePatch, err := patch.AsDeleteGatheringLineByChargeIDPatch()
		if err != nil {
			return Patch{}, err
		}

		if err := deletePatch.RequireCharge(chargeID); err != nil {
			return Patch{}, err
		}

		return patch, nil
	default:
		return Patch{}, fmt.Errorf("expected gathering line by charge patch, got %s", patch.Op())
	}
}

func (p Patches) RequireType(op PatchOperation, countMatcher func(int) error) error {
	if err := countMatcher(len(p)); err != nil {
		return err
	}

	for _, patch := range p {
		if patch.Op() != op {
			return fmt.Errorf("expected %s patch, got %s", op, patch.Op())
		}
	}

	return nil
}

func CountLessThanOrEqualTo(c int) func(int) error {
	return func(count int) error {
		if count > c {
			return fmt.Errorf("expected less than or equal to %d, got %d", c, count)
		}
		return nil
	}
}

func (p Patches) requireSingularPatch(kind string) (Patch, error) {
	if len(p) == 0 {
		return Patch{}, fmt.Errorf("no %s patches provided", kind)
	}

	if len(p) > 1 {
		return Patch{}, fmt.Errorf("expected singular %s patch, got %d", kind, len(p))
	}

	return p[0], nil
}

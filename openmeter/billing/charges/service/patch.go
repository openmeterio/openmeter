package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) ApplyPatches(ctx context.Context, input charges.ApplyPatchesInput) error {
	if input.CustomerID.Namespace != "" && len(input.Creates) > 0 {
		intentsWithDefaults, err := s.applyDefaultTaxCodes(ctx, input.CustomerID.Namespace, input.Creates)
		if err != nil {
			return err
		}

		input.Creates = intentsWithDefaults
	}

	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		if err := s.applyPatches(ctx, input.CustomerID, input.PatchesByChargeID); err != nil {
			return err
		}

		if len(input.Creates) > 0 {
			// Charge creation is the last step as patches might delete a charge whose UniqueReferenceID is used in the creation.
			_, err := s.Create(ctx, charges.CreateInput{
				Namespace: input.CustomerID.Namespace,
				Intents:   input.Creates,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *service) applyPatches(ctx context.Context, customerID customer.CustomerID, patchesByChargeID map[string]charges.Patch) error {
	chargesItems, err := s.adapter.GetByIDs(ctx, charges.GetByIDsInput{
		Namespace: customerID.Namespace,
		IDs:       lo.Keys(patchesByChargeID),
	})
	if err != nil {
		return err
	}

	// Let's validate the charges items
	for _, charge := range chargesItems {
		if charge.CustomerID != customerID.ID {
			return fmt.Errorf("charge %s is not owned by customer %s", charge.ID.ID, customerID.ID)
		}

		if charge.ID.Namespace != customerID.Namespace {
			return fmt.Errorf("charge %s is not in namespace %s, expected %s", charge.ID.ID, charge.ID.Namespace, customerID.Namespace)
		}
	}

	invocableChargesByID, err := s.newInvocableCharges(chargesItems)
	if err != nil {
		return err
	}

	return s.applyInvocableChargePatches(ctx, customerID, invocableChargesByID, patchesByChargeID)
}

// applyInvocableChargePatches advances all affected charges in rounds so invoice
// effects produced at the same lifecycle boundary remain one customer batch.
// Charges that emit effects are reloaded and resumed only after that batch is
// applied; charges that reach stability leave the next round.
func (s *service) applyInvocableChargePatches(
	ctx context.Context,
	customerID customer.CustomerID,
	invocableChargesByID map[string]InvocableCharge,
	patchesByChargeID map[string]charges.Patch,
) error {
	var invoicePatches invoiceupdater.Patches
	pendingAdvancement := make(map[string]InvocableCharge, len(patchesByChargeID))

	for chargeID, patch := range patchesByChargeID {
		invocableCharge, ok := invocableChargesByID[chargeID]
		if !ok {
			return fmt.Errorf("charge %s not found", chargeID)
		}

		result, err := invocableCharge.TriggerPatch(ctx, patch)
		if err != nil {
			return err
		}

		invoicePatches = append(invoicePatches, result.InvoicePatches...)
		if result.CanAdvance {
			if len(result.InvoicePatches) == 0 {
				return fmt.Errorf("charge %s can advance without an invoice-effect boundary", chargeID)
			}
			pendingAdvancement[chargeID] = invocableCharge
		}
	}

	_, err := s.advanceChargesAndApplyInvoicePatches(ctx, customerID, pendingAdvancement, invoicePatches)
	return err
}

// advanceChargesAndApplyInvoicePatches preserves customer-level invoice batches
// while resuming only charges whose preceding effect boundary still permits
// Next. The returned map contains the latest result for each resumed charge.
func (s *service) advanceChargesAndApplyInvoicePatches(
	ctx context.Context,
	customerID customer.CustomerID,
	pendingAdvancement map[string]InvocableCharge,
	invoicePatches invoiceupdater.Patches,
) (map[string]TriggerPatchResult, error) {
	latestResults := make(map[string]TriggerPatchResult, len(pendingAdvancement))

	for len(invoicePatches) > 0 {
		if err := s.invoiceUpdater.ApplyPatches(ctx, customerID, invoicePatches); err != nil {
			return nil, fmt.Errorf("applying invoice patches: %w", err)
		}
		if len(pendingAdvancement) == 0 {
			return latestResults, nil
		}

		invoicePatches = nil
		nextPendingAdvancement := make(map[string]InvocableCharge, len(pendingAdvancement))
		for chargeID, invocableCharge := range pendingAdvancement {
			result, err := invocableCharge.AdvanceCharge(ctx)
			if err != nil {
				return nil, fmt.Errorf("advancing charge %s after invoice patches: %w", chargeID, err)
			}
			latestResults[chargeID] = result

			invoicePatches = append(invoicePatches, result.InvoicePatches...)
			if result.CanAdvance {
				if len(result.InvoicePatches) == 0 {
					return nil, fmt.Errorf("charge %s can advance without an invoice-effect boundary", chargeID)
				}
				nextPendingAdvancement[chargeID] = invocableCharge
			}
		}

		pendingAdvancement = nextPendingAdvancement
	}

	return latestResults, nil
}

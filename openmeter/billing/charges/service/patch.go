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

	var invoicePatches []invoiceupdater.Patch

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
	}

	if len(invoicePatches) > 0 {
		if err := s.invoiceUpdater.ApplyPatches(ctx, customerID, invoicePatches); err != nil {
			return fmt.Errorf("applying invoice patches: %w", err)
		}
	}

	return nil
}

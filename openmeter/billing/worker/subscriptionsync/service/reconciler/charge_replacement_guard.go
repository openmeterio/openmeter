package reconciler

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
)

func (s *Service) ensureChargeCanBeReplaced(ctx context.Context, existing persistedstate.Item) error {
	if s.chargesService == nil || s.billingService == nil {
		return errors.New("charge and billing services are required for the charge replacement guard")
	}

	chargeID := chargesmeta.ChargeID(existing.ID())
	charge, err := s.chargesService.GetByID(ctx, charges.GetByIDInput{
		ChargeID: chargeID,
		Expands:  chargesmeta.Expands{chargesmeta.ExpandRealizations, chargesmeta.ExpandDeletedRealizations},
	})
	if err != nil {
		return fmt.Errorf("loading charge realizations: %w", err)
	}

	var invoiceIDs []string
	switch existing.Type() {
	case persistedstate.ItemTypeChargeFlatFee:
		flatFee, err := charge.AsFlatFeeCharge()
		if err != nil {
			return fmt.Errorf("loading flat fee realizations: %w", err)
		}

		runs := slices.Clone(flatFee.Realizations.PriorRuns)
		if flatFee.Realizations.CurrentRun != nil {
			runs = append(runs, *flatFee.Realizations.CurrentRun)
		}
		invoiceIDs = lo.FilterMap(runs, func(run flatfee.RealizationRun, _ int) (string, bool) {
			if run.InvoiceID == nil {
				return "", false
			}

			return *run.InvoiceID, true
		})
	case persistedstate.ItemTypeChargeUsageBased:
		usageBased, err := charge.AsUsageBasedCharge()
		if err != nil {
			return fmt.Errorf("loading usage based realizations: %w", err)
		}

		invoiceIDs = lo.FilterMap(usageBased.Realizations, func(run usagebased.RealizationRun, _ int) (string, bool) {
			if run.InvoiceID == nil {
				return "", false
			}

			return *run.InvoiceID, true
		})
	default:
		return fmt.Errorf("unsupported charge item type: %s", existing.Type())
	}

	invoiceIDs = lo.Uniq(invoiceIDs)
	if len(invoiceIDs) == 0 {
		return nil
	}
	slices.Sort(invoiceIDs)

	invoices, err := s.billingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
		Namespace:      chargeID.Namespace,
		IDs:            invoiceIDs,
		IncludeDeleted: true,
	})
	if err != nil {
		return fmt.Errorf("loading realization invoices for charge replacement: %w", err)
	}

	requestedIDs := make(map[string]struct{}, len(invoiceIDs))
	for _, id := range invoiceIDs {
		requestedIDs[id] = struct{}{}
	}
	for _, invoice := range invoices.Items {
		if _, ok := requestedIDs[invoice.ID]; !ok {
			return fmt.Errorf("unexpected realization invoice[%s] for charge replacement", invoice.ID)
		}
		delete(requestedIDs, invoice.ID)
	}
	if len(requestedIDs) != 0 {
		return fmt.Errorf("only %d of %d realization invoices were returned", len(invoiceIDs)-len(requestedIDs), len(invoiceIDs))
	}

	for _, invoice := range invoices.Items {
		// ListStandardInvoices returns cached status details without resolving the invoice state machine.
		if lo.IsEmpty(invoice.StatusDetails) {
			return fmt.Errorf("invoice[%s] has no resolved status details", invoice.ID)
		}

		if invoice.StatusDetails.Immutable {
			return fmt.Errorf("invoice[%s] is immutable (status: %s)", invoice.ID, invoice.Status)
		}
	}

	return nil
}

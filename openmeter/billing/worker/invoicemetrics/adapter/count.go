package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/invoicemetrics"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) CountOverdueInvoices(ctx context.Context, input invoicemetrics.CountOverdueInvoicesInput) (invoicemetrics.OverdueInvoiceCounts, error) {
	if err := input.Validate(); err != nil {
		return invoicemetrics.OverdueInvoiceCounts{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (invoicemetrics.OverdueInvoiceCounts, error) {
		cutoff := input.AsOf.Add(-input.MinimumAge)

		collectionQuery := tx.db.BillingInvoice.Query().
			Where(
				billinginvoice.DeletedAtIsNil(),
				billinginvoice.StatusEQ(billing.StandardInvoiceStatusGathering),
				billinginvoice.Or(
					billinginvoice.CollectionAtLTE(cutoff),
					billinginvoice.And(
						billinginvoice.CollectionAtIsNil(),
						billinginvoice.UpdatedAtLTE(cutoff),
					),
				),
			)

		if len(input.ExcludedNamespaces) > 0 {
			collectionQuery.Where(billinginvoice.NamespaceNotIn(input.ExcludedNamespaces...))
		}

		collectionCount, err := collectionQuery.Count(ctx)
		if err != nil {
			return invoicemetrics.OverdueInvoiceCounts{}, fmt.Errorf("failed to count invoices overdue for collection: %w", err)
		}

		advancementCount, err := tx.billingAdapter.CountStandardInvoicesPendingAdvancement(ctx, billing.CountStandardInvoicesPendingAdvancementInput{
			Filter: billing.InvoicePendingAdvancementFilter{
				AsOf:       input.AsOf,
				MinimumAge: input.MinimumAge,
			},
			ExcludedNamespaces: input.ExcludedNamespaces,
		})
		if err != nil {
			return invoicemetrics.OverdueInvoiceCounts{}, fmt.Errorf("failed to count invoices overdue for advancement: %w", err)
		}

		return invoicemetrics.OverdueInvoiceCounts{
			Collection:  int64(collectionCount),
			Advancement: advancementCount,
		}, nil
	})
}

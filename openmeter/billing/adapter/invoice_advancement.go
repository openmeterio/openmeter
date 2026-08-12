package billingadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func invoicePendingAdvancementPredicate(filter billing.InvoicePendingAdvancementFilter) predicate.BillingInvoice {
	cutoff := filter.AsOf.Add(-filter.MinimumAge)

	// Available actions are cached when an invoice is persisted and do not change
	// merely because time passes. The scheduled branches detect newly due states;
	// the cached-action branch remains the fail-safe for other advanceable states.
	return billinginvoice.And(
		billinginvoice.StatusNEQ(billing.StandardInvoiceStatusGathering),
		billinginvoice.Or(
			// The automatic draft approval period has elapsed.
			billinginvoice.And(
				billinginvoice.StatusEQ(billing.StandardInvoiceStatusDraftWaitingAutoApproval),
				billinginvoice.DraftUntilLTE(cutoff),
			),
			// The invoice's quantity collection window has elapsed.
			billinginvoice.And(
				billinginvoice.StatusEQ(billing.StandardInvoiceStatusDraftWaitingForCollection),
				billinginvoice.Or(
					billinginvoice.CollectionAtLTE(cutoff),
					billinginvoice.And(
						billinginvoice.CollectionAtIsNil(),
						billinginvoice.UpdatedAtLTE(cutoff),
					),
				),
			),
			// The state machine exposes another immediately advanceable transition;
			// this is the worker's fail-safe for invoices stuck between stable states.
			billinginvoice.And(
				entutils.JSONBKeyExistsInObject(
					billinginvoice.FieldStatusDetailsCache,
					"availableActions",
					string(billing.InvoiceAvailableActionsFilterAdvance),
				),
				billinginvoice.UpdatedAtLTE(cutoff),
			),
		),
	)
}

func (a *adapter) ListStandardInvoicesPendingAdvancement(ctx context.Context, input billing.ListStandardInvoicesPendingAdvancementInput) ([]billing.InvoiceAdvancementCandidate, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{Err: err}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]billing.InvoiceAdvancementCandidate, error) {
		query := tx.db.BillingInvoice.Query().
			Where(
				billinginvoice.DeletedAtIsNil(),
				invoicePendingAdvancementPredicate(billing.InvoicePendingAdvancementFilter{
					AsOf:       input.AsOf,
					MinimumAge: input.MinimumAge,
				}),
			)

		if len(input.Namespaces) > 0 {
			query.Where(billinginvoice.NamespaceIn(input.Namespaces...))
		}

		if len(input.IDs) > 0 {
			query.Where(billinginvoice.IDIn(input.IDs...))
		}

		query.
			Select(billinginvoice.FieldNamespace, billinginvoice.FieldID, billinginvoice.FieldCustomerID).
			Order(billinginvoice.ByCreatedAt(entutils.GetOrdering(sortx.OrderDefault)...))

		entities, err := query.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list standard invoices pending advancement: %w", err)
		}

		invoices := make([]billing.InvoiceAdvancementCandidate, 0, len(entities))
		for _, entity := range entities {
			invoices = append(invoices, billing.InvoiceAdvancementCandidate{
				InvoiceID: billing.InvoiceID{
					Namespace: entity.Namespace,
					ID:        entity.ID,
				},
				CustomerID: entity.CustomerID,
			})
		}

		return invoices, nil
	})
}

func (a *adapter) CountStandardInvoicesPendingAdvancement(ctx context.Context, input billing.CountStandardInvoicesPendingAdvancementInput) (int64, error) {
	if err := input.Validate(); err != nil {
		return 0, billing.ValidationError{Err: err}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (int64, error) {
		query := tx.db.BillingInvoice.Query().Where(
			billinginvoice.DeletedAtIsNil(),
			invoicePendingAdvancementPredicate(input.Filter),
		)

		if len(input.ExcludedNamespaces) > 0 {
			query.Where(billinginvoice.NamespaceNotIn(input.ExcludedNamespaces...))
		}

		count, err := query.Count(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to count standard invoices pending advancement: %w", err)
		}

		return int64(count), nil
	})
}

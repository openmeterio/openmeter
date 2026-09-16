package billingadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) SetGatheringLineSubscriptionReferenceByChargeID(ctx context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error {
	return a.setLineSubscriptionReferenceByChargeID(ctx, input, billinginvoice.StatusEQ(billing.StandardInvoiceStatusGathering))
}

func (a *adapter) SetStandardLineSubscriptionReferenceByChargeID(ctx context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error {
	return a.setLineSubscriptionReferenceByChargeID(ctx, input, billinginvoice.StatusNEQ(billing.StandardInvoiceStatusGathering))
}

func (a *adapter) setLineSubscriptionReferenceByChargeID(
	ctx context.Context,
	input billing.SetLineSubscriptionReferenceByChargeIDInput,
	invoicePredicate predicate.BillingInvoice,
) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{Err: err}
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		_, err := tx.db.BillingInvoiceLine.Update().
			Where(
				billinginvoiceline.Namespace(input.Namespace),
				billinginvoiceline.ChargeID(input.ChargeID),
				billinginvoiceline.HasBillingInvoiceWith(invoicePredicate),
			).
			SetSubscriptionID(input.SubscriptionID).
			SetSubscriptionPhaseID(input.PhaseID).
			SetSubscriptionItemID(input.ItemID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("updating invoice lines: %w", err)
		}

		return nil
	})
}

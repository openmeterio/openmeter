package billingadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelinediscount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingstandardinvoicedetailedline"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.InvoiceAppAdapter = (*adapter)(nil)

func (a *adapter) SyncExternalIDs(ctx context.Context, in billing.SyncExternalIDsInput) error {
	if err := in.Validate(); err != nil {
		return billing.ValidationError{Err: err}
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		// Update invoice external ID
		if in.InvoicingExternalID != nil {
			_, err := tx.db.BillingInvoice.UpdateOneID(in.Invoice.ID).
				Where(billinginvoice.Namespace(in.Invoice.Namespace)).
				SetInvoicingAppExternalID(*in.InvoicingExternalID).
				Save(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					return fmt.Errorf("invoice not found [id=%s]", in.Invoice.ID)
				}
				return fmt.Errorf("updating invoice external ID: %w", err)
			}
		}

		// Update detailed line external IDs
		for lineID, externalID := range in.LineExternalIDs {
			_, err := tx.db.BillingStandardInvoiceDetailedLine.UpdateOneID(lineID).
				Where(billingstandardinvoicedetailedline.InvoiceID(in.Invoice.ID)).
				SetInvoicingAppExternalID(externalID).
				Save(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					// Line may not exist if invoice structure changed; skip
					continue
				}
				return fmt.Errorf("updating line external ID [lineID=%s]: %w", lineID, err)
			}
		}

		// Update discount external IDs
		for discountID, externalID := range in.LineDiscountExternalIDs {
			_, err := tx.db.BillingInvoiceLineDiscount.UpdateOneID(discountID).
				Where(billinginvoicelinediscount.HasBillingInvoiceLineWith(
					billinginvoiceline.InvoiceID(in.Invoice.ID),
				)).
				SetInvoicingAppExternalID(externalID).
				Save(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					continue
				}
				return fmt.Errorf("updating discount external ID [discountID=%s]: %w", discountID, err)
			}
		}

		return nil
	})
}

func (a *adapter) UpdateInvoiceFields(ctx context.Context, in billing.UpdateInvoiceFieldsInput) error {
	if err := in.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		updateQuery := tx.db.BillingInvoice.UpdateOneID(in.Invoice.ID).
			Where(billinginvoice.Namespace(in.Invoice.Namespace))

		if in.SentToCustomerAt.IsPresent() {
			updateQuery = updateQuery.SetOrClearSentToCustomerAt(in.SentToCustomerAt.OrEmpty())
		}

		_, err := updateQuery.Save(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return fmt.Errorf("invoice not found [id=%s]", in.Invoice.ID)
			}

			return err
		}

		return nil
	})
}

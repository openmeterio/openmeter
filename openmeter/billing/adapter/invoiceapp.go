package billingadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.InvoiceAppAdapter = (*adapter)(nil)

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

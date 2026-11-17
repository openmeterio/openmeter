package appstripeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripeinvoicegenerationdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripeinvoicegeneration"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) GetLatestInvoiceGeneration(ctx context.Context, invoiceID billing.InvoiceID) (int, error) {
	if err := invoiceID.Validate(); err != nil {
		return 0, models.NewGenericValidationError(fmt.Errorf("invalid invoice id: %w", err))
	}

	// Read outside the caller's transaction, the same way the generation is written.
	generation, err := a.db.AppStripeInvoiceGeneration.Query().
		Where(
			appstripeinvoicegenerationdb.Namespace(invoiceID.Namespace),
			appstripeinvoicegenerationdb.InvoiceID(invoiceID.ID),
		).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return 0, nil
		}

		return 0, fmt.Errorf("failed to get stripe invoice generation: %w", err)
	}

	return generation.Generation, nil
}

func (a *adapter) IncrementInvoiceGeneration(ctx context.Context, invoiceID billing.InvoiceID) error {
	if err := invoiceID.Validate(); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("invalid invoice id: %w", err))
	}

	// Not joining the caller's transaction: the Stripe invoice is already given up, and a rolled back
	// invoice sync must not bring back the idempotency key Stripe answers with that invoice.
	err := a.db.AppStripeInvoiceGeneration.Create().
		SetNamespace(invoiceID.Namespace).
		SetInvoiceID(invoiceID.ID).
		SetGeneration(1).
		OnConflictColumns(
			appstripeinvoicegenerationdb.FieldNamespace,
			appstripeinvoicegenerationdb.FieldInvoiceID,
		).
		AddGeneration(1).
		UpdateUpdatedAt().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to increment stripe invoice generation: %w", err)
	}

	return nil
}

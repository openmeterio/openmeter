package billingadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
)

var _ billing.InvoiceLineAdapter = (*adapter)(nil)

func (r *adapter) CreateInvoiceLines(ctx context.Context, input billing.CreateInvoiceLinesAdapterInput) (*billing.CreateInvoiceLinesResponse, error) {
	result := &billing.CreateInvoiceLinesResponse{
		Lines: make([]billingentity.Line, 0, len(input.Lines)),
	}

	for _, line := range input.Lines {
		newEnt := r.db.BillingInvoiceLine.Create().
			SetNamespace(input.Namespace).
			SetInvoiceID(line.InvoiceID).
			SetPeriodStart(line.Period.Start).
			SetPeriodEnd(line.Period.End).
			SetInvoiceAt(line.InvoiceAt).
			SetStatus(line.Status).
			SetType(line.Type).
			SetName(line.Name).
			SetCurrency(line.Currency).
			SetTaxOverrides(line.TaxOverrides).
			SetMetadata(line.Metadata)

		edges := db.BillingInvoiceLineEdges{}

		switch line.Type {
		case billingentity.InvoiceLineTypeManualFee:
			// Let's create the manual line for the invoice
			newManualLineConfig, err := r.db.BillingInvoiceManualLineConfig.Create().
				SetNamespace(input.Namespace).
				SetUnitPrice(line.ManualFee.Price).
				Save(ctx)
			if err != nil {
				return nil, err
			}

			newEnt = newEnt.SetBillingInvoiceManualLines(newManualLineConfig).
				SetQuantity(line.ManualFee.Quantity)

			edges.BillingInvoiceManualLines = newManualLineConfig
		default:
			return nil, fmt.Errorf("unsupported type: %s", line.Type)
		}

		savedLine, err := newEnt.Save(ctx)
		if err != nil {
			return nil, err
		}

		savedLine.Edges = edges

		result.Lines = append(result.Lines, mapInvoiceLineFromDB(savedLine))
	}

	return result, nil
}

func (r *adapter) ListInvoiceLines(ctx context.Context, input billing.ListInvoiceLinesAdapterInput) ([]billingentity.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	query := r.db.BillingInvoiceLine.Query().
		Where(billinginvoiceline.Namespace(input.Namespace))

	if len(input.LineIDs) > 0 {
		query = query.Where(billinginvoiceline.IDIn(input.LineIDs...))
	}

	if input.InvoiceAtBefore != nil {
		query = query.Where(billinginvoiceline.InvoiceAtLT(*input.InvoiceAtBefore))
	}

	query = query.WithBillingInvoice(func(biq *db.BillingInvoiceQuery) {
		biq.Where(billinginvoice.Namespace(input.Namespace))

		if input.CustomerID != "" {
			biq.Where(billinginvoice.CustomerID(input.CustomerID))
		}

		if len(input.InvoiceStatuses) > 0 {
			biq.Where(billinginvoice.StatusIn(input.InvoiceStatuses...))
		}
	})

	dbLines, err := query.
		WithBillingInvoiceManualLines().
		All(ctx)
	if err != nil {
		return nil, err
	}

	return lo.Map(dbLines, func(line *db.BillingInvoiceLine, _ int) billingentity.Line {
		return mapInvoiceLineFromDB(line)
	}), nil
}

func (r *adapter) AssociateLinesToInvoice(ctx context.Context, input billing.AssociateLinesToInvoiceAdapterInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	nAffected, err := r.db.BillingInvoiceLine.Update().
		SetInvoiceID(input.Invoice.ID).
		Where(billinginvoiceline.Namespace(input.Invoice.Namespace)).
		Where(billinginvoiceline.IDIn(input.LineIDs...)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("associating lines: %w", err)
	}

	if nAffected != len(input.LineIDs) {
		return fmt.Errorf("fewer lines were associated (%d) than expected (%d)", nAffected, len(input.LineIDs))
	}

	return nil
}

func mapInvoiceLineFromDB(dbLine *db.BillingInvoiceLine) billingentity.Line {
	invoiceLine := billingentity.Line{
		LineBase: billingentity.LineBase{
			Namespace: dbLine.Namespace,
			ID:        dbLine.ID,

			CreatedAt: dbLine.CreatedAt,
			UpdatedAt: dbLine.UpdatedAt,
			DeletedAt: dbLine.DeletedAt,

			Metadata:  dbLine.Metadata,
			InvoiceID: dbLine.InvoiceID,

			Period: billingentity.Period{
				Start: dbLine.PeriodStart,
				End:   dbLine.PeriodEnd,
			},

			InvoiceAt: dbLine.InvoiceAt,

			Name: dbLine.Name,

			Type:     dbLine.Type,
			Currency: dbLine.Currency,

			TaxOverrides: dbLine.TaxOverrides,
		},
	}

	switch dbLine.Type {
	case billingentity.InvoiceLineTypeManualFee:
		invoiceLine.ManualFee = &billingentity.ManualFeeLine{
			Price:    dbLine.Edges.BillingInvoiceManualLines.UnitPrice,
			Quantity: lo.FromPtrOr(dbLine.Quantity, alpacadecimal.Zero),
		}
	}

	return invoiceLine
}

package billingadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/pkg/models"
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
			SetNillableQuantity(line.Quantity).
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

			newEnt = newEnt.SetBillingInvoiceManualLines(newManualLineConfig)
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

func (r *adapter) GetInvoiceLineByID(ctx context.Context, id models.NamespacedID) (billingentity.Line, error) {
	dbLine, err := r.db.BillingInvoiceLine.Query().
		Where(billinginvoiceline.ID(id.ID)).
		Where(billinginvoiceline.Namespace(id.Namespace)).
		WithBillingInvoiceManualLines().
		Only(ctx)
	if err != nil {
		return billingentity.Line{}, err
	}

	return mapInvoiceLineFromDB(dbLine), nil
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
			Quantity: dbLine.Quantity,
			Currency: dbLine.Currency,

			TaxOverrides: dbLine.TaxOverrides,
		},
	}

	switch dbLine.Type {
	case billingentity.InvoiceLineTypeManualFee:
		invoiceLine.ManualFee = &billingentity.ManualFeeLine{
			Price: dbLine.Edges.BillingInvoiceManualLines.UnitPrice,
		}
	}

	return invoiceLine
}

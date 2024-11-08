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
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ billing.InvoiceLineAdapter = (*adapter)(nil)

func (r *adapter) CreateInvoiceLines(ctx context.Context, input billing.CreateInvoiceLinesAdapterInput) (*billing.CreateInvoiceLinesResponse, error) {
	result := &billing.CreateInvoiceLinesResponse{
		Lines: make([]billingentity.Line, 0, len(input)),
	}

	for _, line := range input {
		if line.Namespace == "" {
			return nil, fmt.Errorf("namespace is required")
		}

		newEnt := r.db.BillingInvoiceLine.Create().
			SetNamespace(line.Namespace).
			SetInvoiceID(line.InvoiceID).
			SetPeriodStart(line.Period.Start).
			SetPeriodEnd(line.Period.End).
			SetNillableParentLineID(line.ParentLineID).
			SetInvoiceAt(line.InvoiceAt).
			SetStatus(line.Status).
			SetType(line.Type).
			SetName(line.Name).
			SetCurrency(line.Currency).
			SetMetadata(line.Metadata)

		if line.TaxConfig != nil {
			newEnt = newEnt.SetTaxConfig(*line.TaxConfig)
		}

		edges := db.BillingInvoiceLineEdges{}

		switch line.Type {
		case billingentity.InvoiceLineTypeFee:
			// Let's create the flat fee line for the invoice
			newFlatFeeLineConfig, err := r.db.BillingInvoiceFlatFeeLineConfig.Create().
				SetNamespace(line.Namespace).
				SetAmount(line.FlatFee.Amount).
				Save(ctx)
			if err != nil {
				return nil, err
			}

			newEnt = newEnt.SetFlatFeeLine(newFlatFeeLineConfig).
				SetQuantity(line.FlatFee.Quantity)

			edges.FlatFeeLine = newFlatFeeLineConfig
		case billingentity.InvoiceLineTypeUsageBased:
			newUBPLine, err := r.createUsageBasedLine(ctx, line.Namespace, line)
			if err != nil {
				return nil, err
			}

			newEnt = newEnt.SetUsageBasedLine(newUBPLine)
			edges.UsageBasedLine = newUBPLine
		default:
			return nil, fmt.Errorf("unsupported type: %s", line.Type)
		}

		savedLine, err := newEnt.Save(ctx)
		if err != nil {
			return nil, err
		}

		if line.ParentLineID != nil {
			// Let's fetch the parent line again
			parentLineQuery := r.db.BillingInvoiceLine.Query().
				Where(billinginvoiceline.Namespace(line.Namespace)).
				Where(billinginvoiceline.ID(*line.ParentLineID))

			parentLineQuery = r.expandLineItems(parentLineQuery)

			parentLine, err := parentLineQuery.First(ctx)
			if err != nil {
				return nil, fmt.Errorf("fetching parent line: %w", err)
			}

			edges.ParentLine = parentLine
		}

		savedLine.Edges = edges

		mappedLine, err := mapInvoiceLineFromDB(savedLine)
		if err != nil {
			return nil, fmt.Errorf("mapping line [id=%s]: %w", savedLine.ID, err)
		}

		result.Lines = append(result.Lines, mappedLine)
	}

	return result, nil
}

func (r *adapter) createUsageBasedLine(ctx context.Context, ns string, line billingentity.Line) (*db.BillingInvoiceUsageBasedLineConfig, error) {
	lineConfig, err := r.db.BillingInvoiceUsageBasedLineConfig.Create().
		SetNamespace(ns).
		SetPriceType(line.UsageBased.Price.Type()).
		SetPrice(&line.UsageBased.Price).
		SetFeatureKey(line.UsageBased.FeatureKey).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return lineConfig, nil
}

func (r *adapter) ListInvoiceLines(ctx context.Context, input billing.ListInvoiceLinesAdapterInput) ([]billingentity.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	query := r.db.BillingInvoice.Query().
		Where(billinginvoice.Namespace(input.Namespace))

	if input.CustomerID != "" {
		query = query.Where(billinginvoice.CustomerID(input.CustomerID))
	}

	if len(input.InvoiceStatuses) > 0 {
		query = query.Where(billinginvoice.StatusIn(input.InvoiceStatuses...))
	}

	query = query.WithBillingInvoiceLines(func(q *db.BillingInvoiceLineQuery) {
		q = q.Where(billinginvoiceline.Namespace(input.Namespace))

		if len(input.LineIDs) > 0 {
			q = q.Where(billinginvoiceline.IDIn(input.LineIDs...))
		}

		if input.InvoiceAtBefore != nil {
			q = q.Where(billinginvoiceline.InvoiceAtLT(*input.InvoiceAtBefore))
		}

		if len(input.ParentLineIDs) > 0 {
			if input.ParentLineIDsIncludeParent {
				q = q.Where(
					billinginvoiceline.Or(
						billinginvoiceline.ParentLineIDIn(input.ParentLineIDs...),
						billinginvoiceline.IDIn(input.ParentLineIDs...),
					),
				)
			} else {
				q = q.Where(billinginvoiceline.ParentLineIDIn(input.ParentLineIDs...))
			}
		}

		if len(input.Statuses) > 0 {
			q = q.Where(billinginvoiceline.StatusIn(input.Statuses...))
		}

		r.expandLineItems(q)
	})

	dbInvoices, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	lines := lo.FlatMap(dbInvoices, func(dbInvoice *db.BillingInvoice, _ int) []*db.BillingInvoiceLine {
		return dbInvoice.Edges.BillingInvoiceLines
	})

	return slicesx.MapWithErr(lines, func(line *db.BillingInvoiceLine) (billingentity.Line, error) {
		return mapInvoiceLineFromDB(line)
	})
}

func (r *adapter) expandLineItems(q *db.BillingInvoiceLineQuery) *db.BillingInvoiceLineQuery {
	return q.WithFlatFeeLine().
		WithUsageBasedLine().
		WithParentLine(func(q *db.BillingInvoiceLineQuery) {
			// We cannot call ourselve here, as it would create an infinite loop
			// but given we are only supporting one level of parent line, we can
			// just expand the parent line here
			q.WithFlatFeeLine().
				WithUsageBasedLine()
		})
}

func (r *adapter) UpdateInvoiceLine(ctx context.Context, input billing.UpdateInvoiceLineAdapterInput) (billingentity.Line, error) {
	if err := input.Validate(); err != nil {
		return billingentity.Line{}, err
	}

	existingLine, err := r.db.BillingInvoiceLine.Query().
		WithFlatFeeLine().
		WithUsageBasedLine().
		Where(billinginvoiceline.Namespace(input.Namespace)).
		Where(billinginvoiceline.ID(input.ID)).
		First(ctx)
	if err != nil {
		return billingentity.Line{}, fmt.Errorf("getting line: %w", err)
	}

	if !existingLine.UpdatedAt.Equal(input.UpdatedAt) {
		return billingentity.Line{}, billingentity.ConflictError{
			ID:     input.ID,
			Entity: billingentity.EntityInvoiceLine,
			Err:    fmt.Errorf("line has been updated since last read"),
		}
	}

	// Let's update the line
	updateLine := r.db.BillingInvoiceLine.UpdateOneID(input.ID).
		SetName(input.Name).
		SetMetadata(input.Metadata).
		SetOrClearDescription(input.Description).
		SetInvoiceID(input.InvoiceID).
		SetPeriodStart(input.Period.Start).
		SetPeriodEnd(input.Period.End).
		SetInvoiceAt(input.InvoiceAt).
		SetNillableParentLineID(input.ParentLineID).
		SetStatus(input.Status).
		SetOrClearTaxConfig(input.TaxConfig)

	edges := db.BillingInvoiceLineEdges{}

	// Let's update the line based on the type
	switch input.Type {
	case billingentity.InvoiceLineTypeFee:
		edges.FlatFeeLine, err = r.updateFlatFeeLine(ctx, existingLine.Edges.FlatFeeLine.ID, input, updateLine)
		if err != nil {
			return billingentity.Line{}, err
		}

		updateLine = updateLine.SetQuantity(input.FlatFee.Quantity)
	case billingentity.InvoiceLineTypeUsageBased:
		edges.UsageBasedLine, err = r.updateUsageBasedLine(ctx, existingLine.Edges.UsageBasedLine.ID, input)
		if err != nil {
			return billingentity.Line{}, err
		}

		updateLine = updateLine.SetOrClearQuantity(input.UsageBased.Quantity)
	default:
		return billingentity.Line{}, fmt.Errorf("unsupported line type: %s", input.Type)
	}

	updatedLine, err := updateLine.Save(ctx)
	if err != nil {
		return billingentity.Line{}, fmt.Errorf("updating line: %w", err)
	}

	if input.ParentLineID != nil {
		// Let's fetch the parent line again
		q := r.db.BillingInvoiceLine.Query().
			Where(billinginvoiceline.Namespace(input.Namespace)).
			Where(billinginvoiceline.ID(*input.ParentLineID))

		q = r.expandLineItems(q)

		parentLine, err := q.First(ctx)
		if err != nil {
			return billingentity.Line{}, fmt.Errorf("fetching parent line: %w", err)
		}

		edges.ParentLine = parentLine
	}

	updatedLine.Edges = edges

	return mapInvoiceLineFromDB(updatedLine)
}

func (r *adapter) updateFlatFeeLine(
	ctx context.Context,
	configId string,
	input billing.UpdateInvoiceLineAdapterInput,
	updateLine *db.BillingInvoiceLineUpdateOne,
) (*db.BillingInvoiceFlatFeeLineConfig, error) {
	updateLine.SetQuantity(input.FlatFee.Quantity)

	updatedConfig, err := r.db.BillingInvoiceFlatFeeLineConfig.UpdateOneID(configId).
		SetAmount(input.FlatFee.Amount).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("updating fee line: %w", err)
	}

	return updatedConfig, nil
}

func (r *adapter) updateUsageBasedLine(
	ctx context.Context,
	configId string,
	input billing.UpdateInvoiceLineAdapterInput,
) (*db.BillingInvoiceUsageBasedLineConfig, error) {
	updatedConfig, err := r.db.BillingInvoiceUsageBasedLineConfig.UpdateOneID(configId).
		SetPriceType(input.UsageBased.Price.Type()).
		SetPrice(&input.UsageBased.Price).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("updating fee line: %w", err)
	}

	return updatedConfig, nil
}

func (r *adapter) AssociateLinesToInvoice(ctx context.Context, input billing.AssociateLinesToInvoiceAdapterInput) ([]billingentity.Line, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	nAffected, err := r.db.BillingInvoiceLine.Update().
		SetInvoiceID(input.Invoice.ID).
		Where(billinginvoiceline.Namespace(input.Invoice.Namespace)).
		Where(billinginvoiceline.IDIn(input.LineIDs...)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("associating lines: %w", err)
	}

	if nAffected != len(input.LineIDs) {
		return nil, fmt.Errorf("not all lines were associated")
	}

	return r.fetchLines(ctx, input.Invoice.Namespace, input.LineIDs)
}

func (r *adapter) fetchLines(ctx context.Context, ns string, lineIDs []string) ([]billingentity.Line, error) {
	query := r.db.BillingInvoiceLine.Query().
		Where(billinginvoiceline.Namespace(ns)).
		Where(billinginvoiceline.IDIn(lineIDs...))

	query = r.expandLineItems(query)

	dbLines, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching lines: %w", err)
	}

	return slicesx.MapWithErr(dbLines, func(line *db.BillingInvoiceLine) (billingentity.Line, error) {
		return mapInvoiceLineFromDB(line)
	})
}

func mapInvoiceLineFromDB(dbLine *db.BillingInvoiceLine) (billingentity.Line, error) {
	invoiceLine := billingentity.Line{
		LineBase: billingentity.LineBase{
			Namespace: dbLine.Namespace,
			ID:        dbLine.ID,

			CreatedAt: dbLine.CreatedAt,
			UpdatedAt: dbLine.UpdatedAt,
			DeletedAt: dbLine.DeletedAt,

			Metadata:  dbLine.Metadata,
			InvoiceID: dbLine.InvoiceID,
			Status:    dbLine.Status,

			Period: billingentity.Period{
				Start: dbLine.PeriodStart,
				End:   dbLine.PeriodEnd,
			},

			ParentLineID: dbLine.ParentLineID,

			InvoiceAt: dbLine.InvoiceAt,

			Name: dbLine.Name,

			Type:     dbLine.Type,
			Currency: dbLine.Currency,

			TaxConfig: lo.EmptyableToPtr(dbLine.TaxConfig),
		},
	}

	if (dbLine.Edges.ParentLine != nil) != (dbLine.ParentLineID != nil) { // XOR
		// This happens if the expandLineItems function is not used, please make sure
		// it's called in all code pathes
		return invoiceLine, fmt.Errorf("inconsistent parent line data")
	}

	if dbLine.Edges.ParentLine != nil {
		parentLine, err := mapInvoiceLineFromDB(dbLine.Edges.ParentLine)
		if err != nil {
			return invoiceLine, fmt.Errorf("mapping parent line: %w", err)
		}

		invoiceLine.ParentLine = &parentLine
	}

	switch dbLine.Type {
	case billingentity.InvoiceLineTypeFee:
		invoiceLine.FlatFee = billingentity.FlatFeeLine{
			Amount:   dbLine.Edges.FlatFeeLine.Amount,
			Quantity: lo.FromPtrOr(dbLine.Quantity, alpacadecimal.Zero),
		}
	case billingentity.InvoiceLineTypeUsageBased:
		ubpLine := dbLine.Edges.UsageBasedLine
		if ubpLine == nil {
			return invoiceLine, fmt.Errorf("manual usage based line is missing")
		}
		invoiceLine.UsageBased = billingentity.UsageBasedLine{
			FeatureKey: ubpLine.FeatureKey,
			Price:      *ubpLine.Price,
			Quantity:   dbLine.Quantity,
		}
	default:
		return invoiceLine, fmt.Errorf("unsupported line type[%s]: %s", dbLine.ID, dbLine.Type)
	}

	return invoiceLine, nil
}

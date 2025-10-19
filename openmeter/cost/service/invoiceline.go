package service

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *service) GetInvoiceLineCost(ctx context.Context, params cost.GetInvoiceLineCostParams) (cost.InvoiceLineCost, error) {
	// Validate params
	if err := params.Validate(); err != nil {
		return cost.InvoiceLineCost{}, err
	}

	namespace := params.InvoiceID.Namespace

	// Get the invoice
	invoice, err := a.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: params.InvoiceID,
		Expand: billing.InvoiceExpand{
			Lines:                       true,
			RecalculateGatheringInvoice: true,
		},
	})
	if err != nil {
		return cost.InvoiceLineCost{}, err
	}

	// Find the line with the feature key
	line, ok := lo.Find(invoice.Lines.OrEmpty(), func(line *billing.Line) bool {
		if line.UsageBased == nil {
			return false
		}

		return line.ID == params.InvoiceLineID
	})
	if !ok {
		return cost.InvoiceLineCost{}, models.NewGenericNotFoundError(
			fmt.Errorf("line not found in invoice: %s", params.InvoiceLineID),
		)
	}

	if line.UsageBased == nil {
		return cost.InvoiceLineCost{}, models.NewGenericConflictError(
			fmt.Errorf("not a usage based line: %s", params.InvoiceLineID),
		)
	}

	// Get the feature
	feature, err := a.FeatureService.GetFeature(ctx, namespace, line.UsageBased.FeatureKey, false)
	if err != nil {
		return cost.InvoiceLineCost{}, err
	}

	if feature.MeterSlug == nil {
		return cost.InvoiceLineCost{}, models.NewGenericConflictError(
			fmt.Errorf("no meter for feature: %s", line.UsageBased.FeatureKey),
		)
	}

	// Get the meter
	met, err := a.MeterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: namespace,
		IDOrSlug:  *feature.MeterSlug,
	})
	if err != nil {
		return cost.InvoiceLineCost{}, err
	}

	// Get the customer
	customer, err := a.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: lo.ToPtr(invoice.CustomerID()),
	})
	if err != nil {
		return cost.InvoiceLineCost{}, err
	}

	if customer == nil {
		return cost.InvoiceLineCost{}, fmt.Errorf("customer cannot be nil")
	}

	// Query the meter
	meterQueryParams := streaming.QueryParams{
		From:           &line.Period.Start,
		To:             &line.Period.End,
		FilterGroupBy:  feature.MeterGroupByFilters,
		FilterCustomer: []streaming.Customer{*customer},
		// We ignore late events because the data is ingested after the invoice is collected
		IgnoreLateEvents: invoice.CollectionAt,
		GroupBy:          lo.FromPtrOr(params.GroupBy, nil),
		WindowSize:       params.WindowSize,
		WindowTimeZone:   params.WindowTimeZone,
	}

	// Get usage for the line
	usageRows, err := a.StreamingConnector.QueryMeter(ctx, namespace, met, meterQueryParams)
	if err != nil {
		return cost.InvoiceLineCost{}, err
	}

	// Get the cost per unit
	costPerUnit := alpacadecimal.NewFromInt(0)

	if !line.UsageBased.Quantity.IsZero() {
		costPerUnit = line.Totals.Amount.Div(*line.UsageBased.Quantity)
	}

	totalInternalCost := alpacadecimal.NewFromInt(0)
	internalCostPerUnit := alpacadecimal.NewFromInt(0)

	if feature.Cost != nil {
		internalCostPerUnit = feature.Cost.PerUnitAmount
	}

	// Calculate the cost for each window
	rows := make([]cost.InvoiceLineCostRow, 0, len(usageRows))

	for _, row := range usageRows {
		usage := alpacadecimal.NewFromFloat(row.Value)
		costAmount := usage.Mul(costPerUnit)

		row := cost.InvoiceLineCostRow{
			WindowStart: row.WindowStart,
			WindowEnd:   row.WindowEnd,
			Usage:       usage,
			Cost:        costAmount,
			CostPerUnit: costPerUnit,
			GroupBy:     row.GroupBy,
		}

		if !internalCostPerUnit.IsZero() {
			internalCost := internalCostPerUnit.Mul(usage)
			totalInternalCost = totalInternalCost.Add(internalCost)
			margin := costAmount.Sub(internalCost)
			marginRate := alpacadecimal.NewFromInt(1).Sub(internalCost.Div(costAmount))

			row.InternalCostPerUnit = lo.ToPtr(internalCostPerUnit)
			row.InternalCost = lo.ToPtr(internalCost)
			row.Margin = lo.ToPtr(margin)
			row.MarginRate = lo.ToPtr(marginRate)
		}

		rows = append(rows, row)
	}

	costTotalAmount := line.Totals.Amount
	usageTotal := alpacadecimal.NewFromInt(0)

	if line.UsageBased.Quantity != nil {
		usageTotal = *line.UsageBased.Quantity
	}

	invoiceLineCost := cost.InvoiceLineCost{
		From:        line.Period.Start,
		To:          line.Period.End,
		Currency:    line.Currency,
		CostPerUnit: costPerUnit,
		Usage:       usageTotal,
		Cost:        costTotalAmount,
		Rows:        rows,
	}

	if !totalInternalCost.IsZero() {
		margin := line.Totals.Amount.Sub(totalInternalCost)
		marginRate := alpacadecimal.NewFromInt(1).Sub(totalInternalCost.Div(costTotalAmount))
		internalCostPerUnit := totalInternalCost.Div(costTotalAmount)

		invoiceLineCost.InternalCost = lo.ToPtr(totalInternalCost)
		invoiceLineCost.InternalCostPerUnit = lo.ToPtr(internalCostPerUnit)
		invoiceLineCost.Margin = lo.ToPtr(margin)
		invoiceLineCost.MarginRate = lo.ToPtr(marginRate)
	}

	return invoiceLineCost, nil
}

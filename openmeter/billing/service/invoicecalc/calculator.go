package invoicecalc

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type invoiceCalculatorsByType struct {
	GatheringInvoice             []Calculation
	GatheringInvoiceWithLiveData []Calculation
	Invoice                      []Calculation
}

var InvoiceCalculations = invoiceCalculatorsByType{
	Invoice: []Calculation{
		WithNoDependencies(StandardInvoiceCollectionAt),
		WithNoDependencies(CalculateDraftUntil),
		WithNoDependencies(CalculateDueAt),
		WithNoDependencies(UpsertDiscountCorrelationIDs),
		RecalculateDetailedLinesAndTotals,
		WithNoDependencies(CalculateInvoicePeriod),
		WithNoDependencies(SnapshotTaxConfigIntoLines),
	},
	GatheringInvoice: []Calculation{
		WithNoDependencies(UpsertDiscountCorrelationIDs),
		WithNoDependencies(GatheringInvoiceCollectionAt),
		WithNoDependencies(CalculateInvoicePeriod),
	},
	// Calculations that should be running on a gathering invoice to populate line items
	GatheringInvoiceWithLiveData: []Calculation{
		WithNoDependencies(UpsertDiscountCorrelationIDs),
		WithNoDependencies(GatheringInvoiceCollectionAt),
		RecalculateDetailedLinesAndTotals,
		WithNoDependencies(CalculateInvoicePeriod),
		WithNoDependencies(SnapshotTaxConfigIntoLines),
		WithNoDependencies(FillGatheringDetailedLineMeta),
	},
}

type (
	Calculation func(*billing.StandardInvoice, CalculationDependencies) error
)

type CalculationDependencies struct {
	FeatureMeters billing.FeatureMeters
}

type Calculator interface {
	Calculate(*billing.StandardInvoice, CalculationDependencies) error
	CalculateGatheringInvoice(*billing.StandardInvoice) error
	CalculateGatheringInvoiceWithLiveData(*billing.StandardInvoice, CalculationDependencies) error
}

type ServiceDependencies interface {
	// RecalculateInvoiceTotals recalculates the totals of an invoice
	RecalculateInvoiceTotals(ctx context.Context, invoice *billing.StandardInvoice) error
}

type calculator struct{}

func New() Calculator {
	return &calculator{}
}

func (c *calculator) Calculate(invoice *billing.StandardInvoice, deps CalculationDependencies) error {
	return c.applyCalculations(invoice, InvoiceCalculations.Invoice, deps)
}

func (c *calculator) applyCalculations(invoice *billing.StandardInvoice, calculators []Calculation, deps CalculationDependencies) error {
	var outErr error
	for _, calc := range calculators {
		err := calc(invoice, deps)
		if err != nil {
			outErr = errors.Join(outErr, err)
		}
	}

	return invoice.MergeValidationIssues(
		billing.ValidationWithComponent(
			billing.ValidationComponentOpenMeter,
			outErr),
		billing.ValidationComponentOpenMeter)
}

func (c *calculator) CalculateGatheringInvoice(invoice *billing.StandardInvoice) error {
	if invoice.Status != billing.StandardInvoiceStatusGathering {
		return errors.New("invoice is not a gathering invoice")
	}

	return c.applyCalculations(invoice, InvoiceCalculations.GatheringInvoice, CalculationDependencies{})
}

func (c *calculator) CalculateGatheringInvoiceWithLiveData(invoice *billing.StandardInvoice, deps CalculationDependencies) error {
	if invoice.Status != billing.StandardInvoiceStatusGathering {
		return errors.New("invoice is not a gathering invoice")
	}

	return c.applyCalculations(invoice, InvoiceCalculations.GatheringInvoiceWithLiveData, deps)
}

func WithNoDependencies(cb func(inv *billing.StandardInvoice) error) Calculation {
	return func(inv *billing.StandardInvoice, _ CalculationDependencies) error {
		return cb(inv)
	}
}

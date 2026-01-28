package invoicecalc

import (
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
	Calculation func(*billing.StandardInvoice, CalculatorDependencies) error
)

type Calculator interface {
	Calculate(*billing.StandardInvoice, CalculatorDependencies) error
	CalculateGatheringInvoice(*billing.StandardInvoice) error
	CalculateGatheringInvoiceWithLiveData(*billing.StandardInvoice, CalculatorDependencies) error
}

type CalculatorDependencies struct {
	FeatureMeters billing.FeatureMeters
}

type calculator struct{}

func New() Calculator {
	return &calculator{}
}

func (c *calculator) Calculate(invoice *billing.StandardInvoice, deps CalculatorDependencies) error {
	return c.applyCalculations(invoice, InvoiceCalculations.Invoice, deps)
}

func (c *calculator) applyCalculations(invoice *billing.StandardInvoice, calculators []Calculation, deps CalculatorDependencies) error {
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

	return c.applyCalculations(invoice, InvoiceCalculations.GatheringInvoice, CalculatorDependencies{})
}

func (c *calculator) CalculateGatheringInvoiceWithLiveData(invoice *billing.StandardInvoice, deps CalculatorDependencies) error {
	if invoice.Status != billing.StandardInvoiceStatusGathering {
		return errors.New("invoice is not a gathering invoice")
	}

	return c.applyCalculations(invoice, InvoiceCalculations.GatheringInvoiceWithLiveData, deps)
}

func WithNoDependencies(cb func(inv *billing.StandardInvoice) error) Calculation {
	return func(inv *billing.StandardInvoice, _ CalculatorDependencies) error {
		return cb(inv)
	}
}

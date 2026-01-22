package invoicecalc

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
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
		CalculateInvoicePeriod,
		SnapshotTaxConfigIntoLines,
	},
	GatheringInvoice: []Calculation{
		WithNoDependencies(UpsertDiscountCorrelationIDs),
		WithNoDependencies(GatheringInvoiceCollectionAt),
		CalculateInvoicePeriod,
	},
	// Calculations that should be running on a gathering invoice to populate line items
	GatheringInvoiceWithLiveData: []Calculation{
		WithNoDependencies(UpsertDiscountCorrelationIDs),
		WithNoDependencies(GatheringInvoiceCollectionAt),
		RecalculateDetailedLinesAndTotals,
		CalculateInvoicePeriod,
		SnapshotTaxConfigIntoLines,
		FillGatheringDetailedLineMeta,
	},
}

type (
	Calculation func(*billing.StandardInvoice, CalculatorDependencies) error
)

type Calculator interface {
	Calculate(*billing.StandardInvoice) error
	CalculateGatheringInvoice(*billing.StandardInvoice) error
	CalculateGatheringInvoiceWithLiveData(*billing.StandardInvoice) error
}

type CalculatorDependencies interface {
	LineService() *lineservice.Service
}

type calculator struct {
	lineService *lineservice.Service
}

type Config struct {
	LineService *lineservice.Service
}

func (c Config) Validate() error {
	if c.LineService == nil {
		return errors.New("line service is required")
	}

	return nil
}

func New(c Config) (Calculator, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &calculator{
		lineService: c.LineService,
	}, nil
}

func (c *calculator) Calculate(invoice *billing.StandardInvoice) error {
	return c.applyCalculations(invoice, InvoiceCalculations.Invoice)
}

func (c *calculator) applyCalculations(invoice *billing.StandardInvoice, calculators []Calculation) error {
	var outErr error
	for _, calc := range calculators {
		err := calc(invoice, c)
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

	return c.applyCalculations(invoice, InvoiceCalculations.GatheringInvoice)
}

func (c *calculator) CalculateGatheringInvoiceWithLiveData(invoice *billing.StandardInvoice) error {
	if invoice.Status != billing.StandardInvoiceStatusGathering {
		return errors.New("invoice is not a gathering invoice")
	}

	return c.applyCalculations(invoice, InvoiceCalculations.GatheringInvoiceWithLiveData)
}

func (c *calculator) LineService() *lineservice.Service {
	return c.lineService
}

func WithNoDependencies(cb func(inv *billing.StandardInvoice) error) Calculation {
	return func(inv *billing.StandardInvoice, _ CalculatorDependencies) error {
		return cb(inv)
	}
}

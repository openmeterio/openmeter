package invoicecalc

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type invoiceCalculatorsByType struct {
	GatheringInvoice             []GatheringInvoiceCalculation
	GatheringInvoiceWithLiveData []StandardInvoiceCalculation
	Invoice                      []StandardInvoiceCalculation
}

var InvoiceCalculations = invoiceCalculatorsByType{
	Invoice: []StandardInvoiceCalculation{
		WithNoDependencies(StandardInvoiceCollectionAt),
		WithNoDependencies(CalculateDraftUntil),
		WithNoDependencies(CalculateDueAt),
		WithNoDependencies(UpsertDiscountCorrelationIDs),
		WithNoDependencies(RecalculateTotals),
		WithNoDependencies(CalculateStandardInvoiceServicePeriod),
		SnapshotTaxConfigIntoLines,
		StampInvoicingAppTaxCodes,
	},
	GatheringInvoice: []GatheringInvoiceCalculation{
		WithNoGatheringDependencies(UpsertGatheringInvoiceDiscountCorrelationIDs),
		GatheringInvoiceCollectionAt,
		WithNoGatheringDependencies(CalculateGatheringInvoiceServicePeriod),
	},
	// Calculations that should be running on a gathering invoice to populate line items
	GatheringInvoiceWithLiveData: []StandardInvoiceCalculation{
		WithNoDependencies(UpsertDiscountCorrelationIDs),
		WithNoDependencies(StandardInvoiceCollectionAt),
		WithNoDependencies(RecalculateTotals),
		WithNoDependencies(CalculateStandardInvoiceServicePeriod),
		SnapshotTaxConfigIntoLines,
		WithNoDependencies(FillGatheringDetailedLineMeta),
	},
}

type (
	StandardInvoiceCalculation  func(*billing.StandardInvoice, StandardInvoiceCalculatorDependencies) error
	GatheringInvoiceCalculation func(*billing.GatheringInvoice, GatheringInvoiceCalculatorDependencies) error
)

type Calculator interface {
	Calculate(*billing.StandardInvoice, StandardInvoiceCalculatorDependencies) error
	CalculateGatheringInvoice(*billing.GatheringInvoice, GatheringInvoiceCalculatorDependencies) error
	CalculateGatheringInvoiceWithLiveData(*billing.StandardInvoice, StandardInvoiceCalculatorDependencies) error
}

type StandardInvoiceCalculatorDependencies struct {
	// DefaultTaxConfig is the invoice default's tax config. The calculator merges it into every
	// line so lines never have to look the default up themselves.
	DefaultTaxConfig *productcatalog.TaxCodeConfig

	// TaxCodes holds the invoice default's Tax Code and every line's Tax Code, soft-deleted ones
	// included, so a line keeps the mapping of a Tax Code deleted after the line referenced it.
	TaxCodes productcatalog.TaxCodesByID

	// InvoicingAppType is the type of the invoice's invoicing app. It is empty only when the
	// caller has no apps expanded (invoice simulation), in which case nothing gets stamped.
	InvoicingAppType app.AppType
}

type GatheringInvoiceCalculatorDependencies struct {
	Collection billing.CollectionConfig
}

type calculator struct{}

func New() Calculator {
	return &calculator{}
}

func (c *calculator) Calculate(invoice *billing.StandardInvoice, deps StandardInvoiceCalculatorDependencies) error {
	return c.applyCalculations(invoice, InvoiceCalculations.Invoice, deps)
}

func (c *calculator) applyCalculations(invoice *billing.StandardInvoice, calculators []StandardInvoiceCalculation, deps StandardInvoiceCalculatorDependencies) error {
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

func (c *calculator) CalculateGatheringInvoiceWithLiveData(invoice *billing.StandardInvoice, deps StandardInvoiceCalculatorDependencies) error {
	if invoice.Status != billing.StandardInvoiceStatusGathering {
		return errors.New("invoice is not a gathering invoice")
	}

	return c.applyCalculations(invoice, InvoiceCalculations.GatheringInvoiceWithLiveData, deps)
}

func (c *calculator) CalculateGatheringInvoice(invoice *billing.GatheringInvoice, deps GatheringInvoiceCalculatorDependencies) error {
	var errs []error

	// Note: GatheringInvoice has no ValidationIssues, so we should just return the error as is
	for _, calc := range InvoiceCalculations.GatheringInvoice {
		err := calc(invoice, deps)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func WithNoDependencies(cb func(inv *billing.StandardInvoice) error) StandardInvoiceCalculation {
	return func(inv *billing.StandardInvoice, _ StandardInvoiceCalculatorDependencies) error {
		return cb(inv)
	}
}

func WithNoGatheringDependencies(cb func(inv *billing.GatheringInvoice) error) GatheringInvoiceCalculation {
	return func(inv *billing.GatheringInvoice, _ GatheringInvoiceCalculatorDependencies) error {
		return cb(inv)
	}
}

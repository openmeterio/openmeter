package invoicecalc

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
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
		RecalculateDetailedLinesAndTotals,
		WithNoDependencies(CalculateStandardInvoiceServicePeriod),
		SnapshotTaxConfigIntoLines,
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
		RecalculateDetailedLinesAndTotals,
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

// TaxCodes is a pre-resolved map of Stripe tax codes keyed by their Stripe code string.
// Built before the calculator runs so that SnapshotTaxConfigIntoLines can stamp both
// TaxCodeID and the entity into each line in a single pass without hitting the DB.
type TaxCodes map[string]taxcode.TaxCode

// Get looks up a TaxCode by Stripe code. Returns nil, false if not present.
func (t TaxCodes) Get(stripeCode string) (*taxcode.TaxCode, bool) {
	if t == nil {
		return nil, false
	}
	tc, ok := t[stripeCode]
	if !ok {
		return nil, false
	}
	return &tc, true
}

type StandardInvoiceCalculatorDependencies struct {
	FeatureMeters feature.FeatureMeters
	RatingService rating.Service
	TaxCodes      TaxCodes
	LineEngines   LineEngineResolver
}

type GatheringInvoiceCalculatorDependencies struct {
	Collection billing.CollectionConfig
}

type LineEngineResolver interface {
	Get(billing.LineEngineType) (billing.LineEngine, error)
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

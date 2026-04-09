package invoicecalc

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

type invoiceCalculatorsByType struct {
	LegacyGatheringInvoice       []Calculation
	GatheringInvoice             []GatheringCalculation
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
		WithNoDependencies(CalculateStandardInvoiceServicePeriod),
		SnapshotTaxConfigIntoLines,
	},
	LegacyGatheringInvoice: []Calculation{
		WithNoDependencies(UpsertDiscountCorrelationIDs),
		WithNoDependencies(LegacyGatheringInvoiceCollectionAt),
		WithNoDependencies(CalculateStandardInvoiceServicePeriod),
	},
	GatheringInvoice: []GatheringCalculation{
		UpsertGatheringInvoiceDiscountCorrelationIDs,
		GatheringInvoiceCollectionAt,
		CalculateGatheringInvoiceServicePeriod,
	},
	// Calculations that should be running on a gathering invoice to populate line items
	GatheringInvoiceWithLiveData: []Calculation{
		WithNoDependencies(UpsertDiscountCorrelationIDs),
		WithNoDependencies(LegacyGatheringInvoiceCollectionAt),
		RecalculateDetailedLinesAndTotals,
		WithNoDependencies(CalculateStandardInvoiceServicePeriod),
		SnapshotTaxConfigIntoLines,
		WithNoDependencies(FillGatheringDetailedLineMeta),
	},
}

type (
	Calculation          func(*billing.StandardInvoice, CalculatorDependencies) error
	GatheringCalculation func(*billing.GatheringInvoice) error
)

type Calculator interface {
	Calculate(*billing.StandardInvoice, CalculatorDependencies) error
	CalculateLegacyGatheringInvoice(*billing.StandardInvoice) error
	CalculateGatheringInvoice(*billing.GatheringInvoice) error
	CalculateGatheringInvoiceWithLiveData(*billing.StandardInvoice, CalculatorDependencies) error
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

type CalculatorDependencies struct {
	FeatureMeters feature.FeatureMeters
	RatingService rating.Service
	TaxCodes      TaxCodes
	LineEngines   LineEngineResolver
}

type LineEngineResolver interface {
	Get(billing.LineEngineType) (billing.LineEngine, error)
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

func (c *calculator) CalculateLegacyGatheringInvoice(invoice *billing.StandardInvoice) error {
	if invoice.Status != billing.StandardInvoiceStatusGathering {
		return errors.New("invoice is not a gathering invoice")
	}

	return c.applyCalculations(invoice, InvoiceCalculations.LegacyGatheringInvoice, CalculatorDependencies{})
}

func (c *calculator) CalculateGatheringInvoiceWithLiveData(invoice *billing.StandardInvoice, deps CalculatorDependencies) error {
	if invoice.Status != billing.StandardInvoiceStatusGathering {
		return errors.New("invoice is not a gathering invoice")
	}

	return c.applyCalculations(invoice, InvoiceCalculations.GatheringInvoiceWithLiveData, deps)
}

func (c *calculator) CalculateGatheringInvoice(invoice *billing.GatheringInvoice) error {
	var errs []error

	// Note: GatheringInvoice has no ValidationIssues, so we should just return the error as is
	for _, calc := range InvoiceCalculations.GatheringInvoice {
		err := calc(invoice)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func WithNoDependencies(cb func(inv *billing.StandardInvoice) error) Calculation {
	return func(inv *billing.StandardInvoice, _ CalculatorDependencies) error {
		return cb(inv)
	}
}

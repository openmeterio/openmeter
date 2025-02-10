package invoicecalc

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
)

var InvoiceCalculations = []Calculation{
	DraftUntilIfMissing,
	RecalculateDetailedLinesAndTotals,
	CalculateInvoicePeriod,
	SnapshotTaxConfigIntoLines,
}

type Calculation func(*billing.Invoice, CalculatorDependencies) error

type Calculator interface {
	Calculate(*billing.Invoice) error
}

type CalculatorDependencies interface {
	LineService() *lineservice.Service
}

type calculator struct {
	calculators []Calculation
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
		calculators: InvoiceCalculations,
		lineService: c.LineService,
	}, nil
}

func (c *calculator) Calculate(invoice *billing.Invoice) error {
	var outErr error
	for _, calc := range InvoiceCalculations {
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

func (c *calculator) LineService() *lineservice.Service {
	return c.lineService
}

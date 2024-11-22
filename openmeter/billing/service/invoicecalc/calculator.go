package invoicecalc

import (
	"errors"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/billing/service/lineservice"
)

var InvoiceCalculations = []Calculation{
	DraftUntilIfMissing,
	RecalculateDetailedLinesAndTotals,
}

type Calculation func(*billingentity.Invoice, CalculatorDependencies) error

type Calculator interface {
	Calculate(*billingentity.Invoice) error
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

func (c *calculator) Calculate(invoice *billingentity.Invoice) error {
	var outErr error
	for _, calc := range InvoiceCalculations {
		err := calc(invoice, c)
		if err != nil {
			outErr = errors.Join(outErr, err)
		}
	}

	return invoice.MergeValidationIssues(
		billingentity.ValidationWithComponent(
			billingentity.ValidationComponentOpenMeter,
			outErr),
		billingentity.ValidationComponentOpenMeter)
}

func (c *calculator) LineService() *lineservice.Service {
	return c.lineService
}

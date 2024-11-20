package billingservice

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/billing/service/calculation"
)

type InvoiceCalculator interface {
	Calculate(*billingentity.Invoice) error
}

type invoiceCalculator struct {
	calculators []InvoiceCalculation
}

func NewInvoiceCalculator() InvoiceCalculator {
	return &invoiceCalculator{
		calculators: InvoiceCalculations,
	}
}

func (c *invoiceCalculator) Calculate(i *billingentity.Invoice) error {
	var outErr error
	for _, calc := range InvoiceCalculations {
		changed, err := calc(i)
		if err != nil {
			outErr = errors.Join(outErr, err)
		}

		if changed {
			// TODO: get rid of this and use the same framework as the line service
			i.Changed = true
		}
	}

	return i.MergeValidationIssues(
		billingentity.ValidationWithComponent(
			billingentity.ValidationComponentOpenMeter,
			outErr),
		billingentity.ValidationComponentOpenMeter)
}

type InvoiceCalculation func(*billingentity.Invoice) (bool, error)

var InvoiceCalculations = []InvoiceCalculation{
	calculation.DraftUntilIfMissing,
	calculation.Totals,
}

type MockableInvoiceCalculator struct {
	upstream InvoiceCalculator
	mock     InvoiceCalculator
}

type mockCalculator struct {
	mock.Mock
}

func (m *mockCalculator) Calculate(i *billingentity.Invoice) error {
	args := m.Called(i)

	// This simulates the same behavior as the calculate method for the original
	// implementation. This way the mock can be used to inject calculation errors
	// as if they were coming from a calculate callback.
	return i.MergeValidationIssues(
		billingentity.ValidationWithComponent(
			billingentity.ValidationComponentOpenMeter,
			args.Error(0)),
		billingentity.ValidationComponentOpenMeter)
}

func NewMockableCalculator(*testing.T) *MockableInvoiceCalculator {
	return &MockableInvoiceCalculator{
		upstream: NewInvoiceCalculator(),
	}
}

func (m *MockableInvoiceCalculator) Calculate(i *billingentity.Invoice) error {
	outErr := m.upstream.Calculate(i)

	if m.mock != nil {
		err := m.mock.Calculate(i)
		if err != nil {
			outErr = errors.Join(outErr, err)
		}
	}

	return outErr
}

func (m *MockableInvoiceCalculator) EnableMock() *mockCalculator {
	mock := &mockCalculator{}
	m.mock = mock

	return mock
}

func (m *MockableInvoiceCalculator) DisableMock() {
	m.mock = nil
}

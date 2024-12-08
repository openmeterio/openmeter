package invoicecalc

import (
	"errors"
	"testing"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type MockableInvoiceCalculator struct {
	upstream Calculator

	mock *mockCalculator
}

type mockCalculator struct {
	calculateResult       mo.Option[error]
	calculateResultCalled bool
}

func (m *mockCalculator) Calculate(i *billing.Invoice) error {
	m.calculateResultCalled = true

	res := m.calculateResult.MustGet()

	// This simulates the same behavior as the calculate method for the original
	// implementation. This way the mock can be used to inject calculation errors
	// as if they were coming from a calculate callback.
	return i.MergeValidationIssues(
		billing.ValidationWithComponent(
			billing.ValidationComponentOpenMeter,
			res),
		billing.ValidationComponentOpenMeter)
}

func (m *mockCalculator) OnCalculate(err error) {
	m.calculateResult = mo.Some(err)
}

func (m *mockCalculator) AssertExpectations(t *testing.T) {
	t.Helper()

	if m.calculateResult.IsPresent() && !m.calculateResultCalled {
		t.Errorf("expected Calculate to be called")
	}
}

func (m *mockCalculator) Reset(t *testing.T) {
	t.Helper()

	m.AssertExpectations(t)

	m.calculateResult = mo.None[error]()
	m.calculateResultCalled = false
}

func NewMockableCalculator(_ *testing.T, upstream Calculator) *MockableInvoiceCalculator {
	return &MockableInvoiceCalculator{
		upstream: upstream,
	}
}

func (m *MockableInvoiceCalculator) Calculate(i *billing.Invoice) error {
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

func (m *MockableInvoiceCalculator) DisableMock(t *testing.T) {
	m.mock.AssertExpectations(t)
	m.mock = nil
}

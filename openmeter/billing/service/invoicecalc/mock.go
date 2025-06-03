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

	calculateGatheringInvoiceResult       mo.Option[error]
	calculateGatheringInvoiceResultCalled bool

	calculateGatheringInvoiceWithLiveDataResult       mo.Option[error]
	calculateGatheringInvoiceWithLiveDataResultCalled bool
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

func (m *mockCalculator) CalculateGatheringInvoice(i *billing.Invoice) error {
	m.calculateGatheringInvoiceResultCalled = true

	res := m.calculateGatheringInvoiceResult.MustGet()

	// This simulates the same behavior as the calculate method for the original
	// implementation. This way the mock can be used to inject calculation errors
	// as if they were coming from a calculate callback.
	return i.MergeValidationIssues(
		billing.ValidationWithComponent(
			billing.ValidationComponentOpenMeter,
			res),
		billing.ValidationComponentOpenMeter)
}

func (m *mockCalculator) CalculateGatheringInvoiceWithLiveData(i *billing.Invoice) error {
	m.calculateGatheringInvoiceWithLiveDataResultCalled = true

	res := m.calculateGatheringInvoiceWithLiveDataResult.MustGet()

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

func (m *mockCalculator) OnCalculateGatheringInvoice(err error) {
	m.calculateGatheringInvoiceResult = mo.Some(err)
}

func (m *mockCalculator) OnCalculateGatheringInvoiceWithLiveData(err error) {
	m.calculateGatheringInvoiceWithLiveDataResult = mo.Some(err)
}

func (m *mockCalculator) AssertExpectations(t *testing.T) {
	t.Helper()

	if m.calculateResult.IsPresent() && !m.calculateResultCalled {
		t.Errorf("expected Calculate to be called")
	}

	if m.calculateGatheringInvoiceResult.IsPresent() && !m.calculateGatheringInvoiceResultCalled {
		t.Errorf("expected CalculateGatheringInvoice to be called")
	}

	if m.calculateGatheringInvoiceWithLiveDataResult.IsPresent() && !m.calculateGatheringInvoiceWithLiveDataResultCalled {
		t.Errorf("expected CalculateGatheringInvoiceWithLiveData to be called")
	}
}

func (m *mockCalculator) Reset(t *testing.T) {
	t.Helper()

	m.AssertExpectations(t)

	m.calculateResult = mo.None[error]()
	m.calculateResultCalled = false

	m.calculateGatheringInvoiceResult = mo.None[error]()
	m.calculateGatheringInvoiceResultCalled = false

	m.calculateGatheringInvoiceWithLiveDataResult = mo.None[error]()
	m.calculateGatheringInvoiceWithLiveDataResultCalled = false
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

func (m *MockableInvoiceCalculator) CalculateGatheringInvoice(i *billing.Invoice) error {
	outErr := m.upstream.CalculateGatheringInvoice(i)

	if m.mock != nil {
		err := m.mock.CalculateGatheringInvoice(i)
		if err != nil {
			outErr = errors.Join(outErr, err)
		}
	}

	return outErr
}

func (m *MockableInvoiceCalculator) CalculateGatheringInvoiceWithLiveData(i *billing.Invoice) error {
	outErr := m.upstream.CalculateGatheringInvoiceWithLiveData(i)

	if m.mock != nil {
		err := m.mock.CalculateGatheringInvoiceWithLiveData(i)
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

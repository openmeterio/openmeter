package billing

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestListInvoicesInputValidateRequiresNamespace(t *testing.T) {
	// given:
	// - a list invoices input without a namespace
	// when:
	// - the input is validated
	// then:
	// - validation rejects the request
	err := (ListInvoicesInput{}).Validate()

	require.ErrorContains(t, err, "namespace is required")
}

func TestListInvoicesAdapterInputValidateRequiresNamespace(t *testing.T) {
	// given:
	// - an adapter list input without a namespace
	// when:
	// - the input is validated
	// then:
	// - validation rejects the query
	err := (ListInvoicesAdapterInput{}).Validate()

	require.ErrorContains(t, err, "namespace is required")
}

func TestSimulateInvoiceInputValidateRejectsLineWithoutEngine(t *testing.T) {
	// given:
	// - a simulation input with one line without an engine and one with an unknown engine
	// when:
	// - the input is validated
	// then:
	// - validation reports every bad line as a generic validation error, so the API responds with 400
	input := SimulateInvoiceInput{
		Namespace:  "ns",
		CustomerID: lo.ToPtr("customer"),
		Currency:   "USD",
		Lines: NewStandardInvoiceLines([]*StandardLine{
			{},
			{StandardLineBase: StandardLineBase{Engine: "unknown"}},
		}),
	}

	err := input.Validate()

	require.ErrorContains(t, err, "line[0]: invalid line engine type")
	require.ErrorContains(t, err, "line[1]: invalid line engine type: unknown")
	require.True(t, models.IsGenericValidationError(err))
}

func TestSimulateInvoiceInputValidateSkipsDeletedLineWithoutEngine(t *testing.T) {
	// given:
	// - a simulation input with a valid line and a deleted line without an engine
	// when:
	// - the input is validated
	// then:
	// - validation succeeds, because simulation ignores deleted lines
	input := SimulateInvoiceInput{
		Namespace:  "ns",
		CustomerID: lo.ToPtr("customer"),
		Currency:   "USD",
		Lines: NewStandardInvoiceLines([]*StandardLine{
			{StandardLineBase: StandardLineBase{Engine: LineEngineTypeInvoice}},
			{StandardLineBase: StandardLineBase{
				ManagedResource: models.ManagedResource{
					ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(time.Now())},
				},
			}},
		}),
	}

	require.NoError(t, input.Validate())
}

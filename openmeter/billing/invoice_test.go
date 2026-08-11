package billing

import (
	"testing"

	"github.com/stretchr/testify/require"
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

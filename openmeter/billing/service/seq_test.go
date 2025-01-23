package billingservice

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomerPrefix(t *testing.T) {
	require.Equal(t, "UNKN", getCustomerPrefix(""))

	require.Equal(t, "JOHN", getCustomerPrefix("John"))
	require.Equal(t, "JO", getCustomerPrefix("Jo"))

	require.Equal(t, "PETU", getCustomerPrefix("Peter Turi"))
	require.Equal(t, "PTU", getCustomerPrefix("P Turi"))
}

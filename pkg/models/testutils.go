package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// FIXME: dependency directions are messed up due to using same lexical package name across tests and package
func RequireValidationIssuesMatch(t *testing.T, expected ValidationIssues, actual ValidationIssues) {
	t.Helper()

	expCopy := expected.Clone()
	actCopy := actual.Clone()

	require.Len(t, actCopy, len(expCopy), "issue count must match")
	for i := range actCopy {
		actualField := actCopy[i].Field()
		expectedField := expCopy[i].Field()

		require.Equal(t, expectedField.String(), actualField.String(), "[code = %s] field string must match at index %d", actual[i].Code(), i)
		require.Equal(t, expectedField.JSONPath(), actualField.JSONPath(), "[code = %s] field json path must match at index %d", actual[i].Code(), i)

		// Do not strip attributes; compare full set, including status code

		actCopy[i].field = nil
		expCopy[i].field = nil
	}

	require.Equalf(t, expCopy, actCopy, "issues must match")
}

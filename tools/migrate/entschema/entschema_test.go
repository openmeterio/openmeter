package entschema_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/tools/migrate/entschema"
)

func TestGenerateSQL_IndependentSchemaSources(t *testing.T) {
	// given: two independent Ent schemas with source-specific supplemental DDL.
	const supplementalDDL = `COMMENT ON TABLE "example1s" IS 'schema-specific DDL';`

	for _, tc := range []struct {
		name        string
		input       entschema.GenerateSQLInput
		expectedSQL []string
		excludedSQL []string
	}{
		{
			name: "first schema with supplemental DDL",
			input: entschema.GenerateSQLInput{
				SchemaPath:      "../../../pkg/framework/entutils/testutils/ent1/schema",
				PostgresVersion: "15",
				SupplementalDDL: supplementalDDL,
			},
			expectedSQL: []string{`CREATE TABLE "example1s"`, `"example_value_1"`, supplementalDDL},
			excludedSQL: []string{`"example2s"`, `"example_value_2"`},
		},
		{
			name: "second schema without supplemental DDL",
			input: entschema.GenerateSQLInput{
				SchemaPath:      "../../../pkg/framework/entutils/testutils/ent2/schema",
				PostgresVersion: "15",
			},
			expectedSQL: []string{`CREATE TABLE "example2s"`, `"example_value_2"`},
			excludedSQL: []string{`"example1s"`, `"example_value_1"`, supplementalDDL},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when: the reusable exporter is configured for this source.
			ddl, err := entschema.GenerateSQL(t.Context(), tc.input)
			require.NoError(t, err)

			// then: output contains only this source's schema and supplemental DDL.
			for _, expected := range tc.expectedSQL {
				require.Contains(t, ddl, expected)
			}

			for _, excluded := range tc.excludedSQL {
				require.NotContains(t, ddl, excluded)
			}
		})
	}
}

func TestGenerateSQL_InvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name          string
		input         entschema.GenerateSQLInput
		expectedError string
	}{
		{
			name:          "missing schema path",
			input:         entschema.GenerateSQLInput{PostgresVersion: "15"},
			expectedError: "schema path is required",
		},
		{
			name:          "missing postgres version",
			input:         entschema.GenerateSQLInput{SchemaPath: "../../../pkg/framework/entutils/testutils/ent1/schema"},
			expectedError: "postgres version is required",
		},
		{
			name:          "missing schema package",
			input:         entschema.GenerateSQLInput{SchemaPath: "testdata/missing", PostgresVersion: "15"},
			expectedError: "load ent schema",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := entschema.GenerateSQL(t.Context(), tc.input)
			require.ErrorContains(t, err, tc.expectedError)
		})
	}
}

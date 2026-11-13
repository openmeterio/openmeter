package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/tools/migrate/entschema"
)

func main() {
	cmd := &cobra.Command{
		Use: "entschema",
		RunE: func(cmd *cobra.Command, _ []string) error {
			extensions, err := os.ReadFile("tools/migrate/extensions.sql")
			if err != nil {
				return err
			}

			supplementalDDL, err := os.ReadFile("tools/migrate/supplemental_schema.sql")
			if err != nil {
				return err
			}

			ddl, err := entschema.GenerateSQL(cmd.Context(), entschema.GenerateSQLInput{
				SchemaPath:      "./openmeter/ent/schema",
				PostgresVersion: "15",
				SupplementalDDL: string(supplementalDDL),
			})
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), string(extensions)+"\n"+ddl)

			return err
		},
	}
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

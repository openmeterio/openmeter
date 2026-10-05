package main

import (
	"fmt"
	"os"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use: "entschema",
		RunE: func(cmd *cobra.Command, _ []string) error {
			graph, err := entc.LoadGraph("./openmeter/ent/schema", &gen.Config{})
			if err != nil {
				return err
			}
			tables, err := graph.Tables()
			if err != nil {
				return err
			}
			views, err := graph.Views()
			if err != nil {
				return err
			}
			ddl, err := schema.DDL(cmd.Context(), schema.DDLArgs{
				Dialect: dialect.Postgres,
				Version: "15",
				Tables:  append(tables, views...),
			})
			if err != nil {
				return err
			}

			// Ent cannot express deferred FKs. Include their timing in Atlas's
			// desired schema so future diffs preserve it.
			constraints, err := os.ReadFile("tools/migrate/deferred_constraints.sql")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n%s", ddl, constraints)
			return err
		},
	}
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

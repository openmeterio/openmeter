package entschema

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"

	"github.com/openmeterio/openmeter/pkg/models"
)

type GenerateSQLInput struct {
	SchemaPath      string
	PostgresVersion string
	// SupplementalDDL preserves database features that Ent cannot express.
	SupplementalDDL string
}

var _ models.Validator = GenerateSQLInput{}

func (i GenerateSQLInput) Validate() error {
	var errs []error

	if i.SchemaPath == "" {
		errs = append(errs, errors.New("schema path is required"))
	}

	if i.PostgresVersion == "" {
		errs = append(errs, errors.New("postgres version is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GenerateSQL exports one Ent schema's tables and views as PostgreSQL DDL,
// followed by any supplemental DDL for that schema source.
func GenerateSQL(ctx context.Context, input GenerateSQLInput) (string, error) {
	if err := input.Validate(); err != nil {
		return "", err
	}

	graph, err := entc.LoadGraph(input.SchemaPath, &gen.Config{})
	if err != nil {
		return "", fmt.Errorf("load ent schema: %w", err)
	}

	tables, err := graph.Tables()
	if err != nil {
		return "", fmt.Errorf("schema tables: %w", err)
	}

	views, err := graph.Views()
	if err != nil {
		return "", fmt.Errorf("schema views: %w", err)
	}

	ddl, err := schema.DDL(ctx, schema.DDLArgs{
		Dialect: dialect.Postgres,
		Version: input.PostgresVersion,
		Tables:  append(tables, views...),
	})
	if err != nil {
		return "", fmt.Errorf("schema DDL: %w", err)
	}

	if input.SupplementalDDL != "" {
		ddl += "\n" + input.SupplementalDDL
	}

	return ddl, nil
}

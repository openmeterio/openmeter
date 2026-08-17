package pgschemadiff

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	pgdiff "github.com/stripe/pg-schema-diff/pkg/diff"
	"github.com/stripe/pg-schema-diff/pkg/tempdb"

	entmigrate "github.com/openmeterio/openmeter/openmeter/ent/db/migrate"
	"github.com/openmeterio/openmeter/pkg/models"
	ommigrate "github.com/openmeterio/openmeter/tools/migrate"
	"github.com/openmeterio/openmeter/tools/migrate/namespacefks"
	"github.com/openmeterio/openmeter/tools/migrate/viewgen"
)

const temporaryDatabaseMaxConnections = 5

var publicSchemaQualifier = strings.NewReplacer(
	`"public".`, "",
	"public.", "",
)

type GeneratePlanInput struct {
	DevDatabaseURL       string
	EntSchemaPath        string
	NamespaceChildTables []string
	SkipPlanValidation   bool
	Logger               *slog.Logger
}

func (i GeneratePlanInput) Validate() error {
	var errs []error

	if strings.TrimSpace(i.DevDatabaseURL) == "" {
		errs = append(errs, errors.New("dev database URL is required"))
	}
	if strings.TrimSpace(i.EntSchemaPath) == "" {
		errs = append(errs, errors.New("ent schema path is required"))
	}
	if i.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GeneratePlan compares the existing migration history with a disposable
// database reconciled to the Ent schema, namespace foreign keys, and Ent views.
// The configured PostgreSQL instance is used only for disposable databases.
func GeneratePlan(ctx context.Context, input GeneratePlanInput) (_ pgdiff.Plan, retErr error) {
	if err := input.Validate(); err != nil {
		return pgdiff.Plan{}, fmt.Errorf("invalid pg-schema-diff input: %w", err)
	}

	baseConfig, err := pgx.ParseConfig(input.DevDatabaseURL)
	if err != nil {
		return pgdiff.Plan{}, fmt.Errorf("parse dev database URL: %w", err)
	}

	factory, err := tempdb.NewOnInstanceFactory(
		ctx,
		newConnectionPoolFactory(baseConfig),
		tempdb.WithRootDatabase(baseConfig.Database),
	)
	if err != nil {
		return pgdiff.Plan{}, fmt.Errorf("create temporary database factory: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, factory.Close())
	}()

	currentDatabase, err := createMigratedDatabase(ctx, factory, input.DevDatabaseURL, input.Logger)
	if err != nil {
		return pgdiff.Plan{}, fmt.Errorf("create current migration database: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, currentDatabase.Close(ctx))
	}()

	desiredDatabase, err := createMigratedDatabase(ctx, factory, input.DevDatabaseURL, input.Logger)
	if err != nil {
		return pgdiff.Plan{}, fmt.Errorf("create desired migration database: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, desiredDatabase.Close(ctx))
	}()

	if err := reconcileDesiredDatabase(ctx, desiredDatabase.ConnPool, input); err != nil {
		return pgdiff.Plan{}, fmt.Errorf("reconcile desired database: %w", err)
	}

	options := []pgdiff.PlanOpt{
		pgdiff.WithTempDbFactory(factory),
		pgdiff.WithIncludeSchemas("public"),
	}
	if input.SkipPlanValidation {
		options = append(options, pgdiff.WithDoNotValidatePlan())
	}

	plan, err := pgdiff.Generate(
		ctx,
		pgdiff.DBSchemaSource(currentDatabase.ConnPool),
		pgdiff.DBSchemaSource(desiredDatabase.ConnPool),
		options...,
	)
	if err != nil {
		return pgdiff.Plan{}, fmt.Errorf("generate schema diff: %w", err)
	}

	return plan, nil
}

func newConnectionPoolFactory(baseConfig *pgx.ConnConfig) tempdb.CreateConnPoolForDbFn {
	return func(ctx context.Context, databaseName string) (*sql.DB, error) {
		config := baseConfig.Copy()
		config.Database = databaseName

		database := stdlib.OpenDB(*config)
		database.SetMaxOpenConns(temporaryDatabaseMaxConnections)
		if err := database.PingContext(ctx); err != nil {
			_ = database.Close()
			return nil, fmt.Errorf("ping database %q: %w", databaseName, err)
		}

		return database, nil
	}
}

func createMigratedDatabase(ctx context.Context, factory tempdb.Factory, devDatabaseURL string, logger *slog.Logger) (_ *tempdb.Database, retErr error) {
	database, err := factory.Create(ctx)
	if err != nil {
		return nil, fmt.Errorf("create temporary database: %w", err)
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, database.Close(ctx))
		}
	}()

	var databaseName string
	if err := database.ConnPool.QueryRowContext(ctx, "SELECT current_database()").Scan(&databaseName); err != nil {
		return nil, fmt.Errorf("query temporary database name: %w", err)
	}

	temporaryDatabaseURL, err := databaseURL(devDatabaseURL, databaseName)
	if err != nil {
		return nil, fmt.Errorf("build temporary database URL: %w", err)
	}
	migrator, err := ommigrate.New(ommigrate.MigrateOptions{
		ConnectionString: temporaryDatabaseURL,
		Migrations:       ommigrate.OMMigrationsConfig,
		Logger:           logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}
	defer migrator.CloseOrLogError()

	if err := migrator.Up(); err != nil {
		return nil, fmt.Errorf("apply migration history: %w", err)
	}
	if _, err := database.ConnPool.ExecContext(ctx, `DROP TABLE IF EXISTS "schema_om"`); err != nil {
		return nil, fmt.Errorf("exclude migration state table: %w", err)
	}

	return database, nil
}

func databaseURL(baseURL, databaseName string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse database URL: %w", err)
	}
	if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
		return "", fmt.Errorf("unsupported database URL scheme %q", parsedURL.Scheme)
	}

	parsedURL.Path = "/" + databaseName
	return parsedURL.String(), nil
}

func reconcileDesiredDatabase(ctx context.Context, database *sql.DB, input GeneratePlanInput) error {
	driver := entsql.OpenDB(dialect.Postgres, database)
	if err := entmigrate.NewSchema(driver).Create(
		ctx,
		entmigrate.WithDropColumn(true),
		entmigrate.WithDropIndex(true),
		entmigrate.WithForeignKeys(true),
	); err != nil {
		return fmt.Errorf("apply Ent schema: %w", err)
	}

	namespaceSQL, err := namespacefks.Generate(namespacefks.GenerateInput{
		Tables:      entmigrate.Tables,
		ChildTables: input.NamespaceChildTables,
	})
	if err != nil {
		return fmt.Errorf("generate namespace foreign keys: %w", err)
	}
	if _, err := database.ExecContext(ctx, string(namespaceSQL)); err != nil {
		return fmt.Errorf("apply namespace foreign keys: %w", err)
	}

	viewsSQL, err := viewgen.GenerateRecreateSQL(input.EntSchemaPath)
	if err != nil {
		return fmt.Errorf("generate Ent views: %w", err)
	}
	if _, err := database.ExecContext(ctx, string(viewsSQL)); err != nil {
		return fmt.Errorf("recreate Ent views: %w", err)
	}

	return nil
}

func RenderSQL(plan pgdiff.Plan, validated bool) []byte {
	var output bytes.Buffer
	output.WriteString("-- Generated by the pg-schema-diff proof of concept. Review before use.\n")
	if validated {
		output.WriteString("-- Replay validation: passed.\n")
	} else {
		output.WriteString("-- Replay validation: disabled.\n")
	}
	fmt.Fprintf(&output, "-- Current schema hash: %s\n", plan.CurrentSchemaHash)
	if len(plan.Statements) == 0 {
		output.WriteString("-- Schema matches the desired state.\n")
		return output.Bytes()
	}

	for index, statement := range plan.Statements {
		output.WriteString("\n/*\n")
		fmt.Fprintf(&output, "Statement %d\n", index)
		for _, hazard := range statement.Hazards {
			fmt.Fprintf(&output, "  - %s: %s\n", hazard.Type, hazard.Message)
		}
		output.WriteString("*/\n")
		fmt.Fprintf(&output, "SET SESSION statement_timeout = %d;\n", statement.Timeout.Milliseconds())
		fmt.Fprintf(&output, "SET SESSION lock_timeout = %d;\n", statement.LockTimeout.Milliseconds())
		ddl := publicSchemaQualifier.Replace(statement.DDL)
		output.WriteString(strings.TrimSuffix(strings.TrimSpace(ddl), ";"))
		output.WriteString(";\n")
	}

	return output.Bytes()
}

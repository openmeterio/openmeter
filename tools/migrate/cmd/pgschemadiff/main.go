package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/openmeterio/openmeter/tools/migrate/pgschemadiff"
)

const defaultNamespaceChildTables = "billing_profiles,billing_customer_overrides,billing_invoices"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("pgschemadiff", flag.ContinueOnError)
	flags.SetOutput(stderr)

	devDatabaseURL := flags.String("dev-dsn", "postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable", "PostgreSQL instance on which disposable databases can be created")
	entSchemaPath := flags.String("ent-schema", "./openmeter/ent/schema", "path to the Ent schema package")
	namespaceChildTables := flags.String("namespace-fk-child-tables", defaultNamespaceChildTables, "comma-separated child tables for generated namespace foreign keys")
	skipPlanValidation := flags.Bool("skip-plan-validation", false, "skip replaying the generated plan against a disposable database")
	outputPath := flags.String("output", "-", "output SQL file, or - for stdout")
	if err := flags.Parse(args); err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	plan, err := pgschemadiff.GeneratePlan(ctx, pgschemadiff.GeneratePlanInput{
		DevDatabaseURL:       *devDatabaseURL,
		EntSchemaPath:        *entSchemaPath,
		NamespaceChildTables: splitCommaSeparated(*namespaceChildTables),
		SkipPlanValidation:   *skipPlanValidation,
		Logger:               logger,
	})
	if err != nil {
		return err
	}

	output := pgschemadiff.RenderSQL(plan, !*skipPlanValidation)
	if *outputPath == "-" {
		_, err := stdout.Write(output)
		return err
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(*outputPath, output, 0o644); err != nil {
		return fmt.Errorf("write schema diff: %w", err)
	}

	return nil
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		values = append(values, value)
	}

	return values
}

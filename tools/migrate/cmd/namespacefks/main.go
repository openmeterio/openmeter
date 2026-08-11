package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	entmigrate "github.com/openmeterio/openmeter/openmeter/ent/db/migrate"
	"github.com/openmeterio/openmeter/tools/migrate/namespacefks"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		childTables string
		check       bool
		output      string
	)

	flag.StringVar(&childTables, "child-tables", "", "comma-separated child tables to include; empty includes all eligible tables")
	flag.BoolVar(&check, "check", false, "fail when the generated SQL differs from the output file")
	flag.StringVar(&output, "output", "-", "output file, or - for stdout")
	flag.Parse()

	generated, err := namespacefks.Generate(namespacefks.GenerateInput{
		Tables:      entmigrate.Tables,
		ChildTables: splitCommaSeparated(childTables),
	})
	if err != nil {
		return err
	}

	if output == "-" {
		if check {
			return errors.New("check requires an output file")
		}

		_, err := os.Stdout.Write(generated)
		return err
	}

	if check {
		existing, err := os.ReadFile(output)
		if err != nil {
			return fmt.Errorf("read generated namespace foreign keys: %w", err)
		}

		if !bytes.Equal(existing, generated) {
			return fmt.Errorf("%s is stale; run make generate-namespace-fks", output)
		}

		return nil
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	temporary, err := os.CreateTemp(filepath.Dir(output), ".namespace-fks-*.sql")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set temporary output permissions: %w", err)
	}

	if _, err := temporary.Write(generated); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary output: %w", err)
	}

	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}

	if err := os.Rename(temporaryName, output); err != nil {
		return fmt.Errorf("replace generated namespace foreign keys: %w", err)
	}

	return nil
}

func splitCommaSeparated(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	tables := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		table := strings.TrimSpace(part)
		if table == "" {
			continue
		}
		if _, ok := seen[table]; ok {
			continue
		}

		seen[table] = struct{}{}
		tables = append(tables, table)
	}

	return tables
}

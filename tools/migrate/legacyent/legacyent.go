package legacyent

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
)

const (
	// BaselineCommit is the OpenMeter commit whose generated Ent schema is frozen in this package.
	BaselineCommit = "12ab7b082035f2f93972c7f98973c5502107c157"
	// BaselineVersion is the latest versioned migration represented by the frozen Ent schema.
	BaselineVersion uint = 20260709134422
)

//go:embed reconciliation/*.sql
var reconciliationFS embed.FS

// MigrateToBaseline applies the additive Ent schema migration behavior frozen at BaselineCommit.
// It deliberately uses Ent's default migration options: foreign keys are enabled, while dropping
// columns and indexes is disabled.
func MigrateToBaseline(ctx context.Context, db *sql.DB) error {
	driver := entsql.OpenDB(dialect.Postgres, db)
	migrator, err := schema.NewMigrate(driver)
	if err != nil {
		return fmt.Errorf("create frozen Ent migrator: %w", err)
	}

	if err := migrator.Create(ctx, Tables...); err != nil {
		return fmt.Errorf("apply frozen Ent schema from commit %s: %w", BaselineCommit, err)
	}

	return nil
}

// Reconcile applies database objects and state that are required at BaselineVersion but are not
// represented by Ent's generated table descriptors. Every script must be safe to run again when
// adoption was interrupted before the migration version was recorded.
func Reconcile(ctx context.Context, db *sql.DB) error {
	entries, err := fs.ReadDir(reconciliationFS, "reconciliation")
	if err != nil {
		return fmt.Errorf("read reconciliation scripts: %w", err)
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := "reconciliation/" + entry.Name()
		contents, err := fs.ReadFile(reconciliationFS, path)
		if err != nil {
			return fmt.Errorf("read reconciliation script %s: %w", entry.Name(), err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin reconciliation script %s: %w", entry.Name(), err)
		}

		if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute reconciliation script %s: %w", entry.Name(), err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit reconciliation script %s: %w", entry.Name(), err)
		}
	}

	return nil
}

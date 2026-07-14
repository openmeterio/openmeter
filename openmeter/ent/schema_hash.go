package entdb

import _ "embed"

//go:embed db/migrate/schema.go
var generatedMigrationSchema string

// GeneratedMigrationSchema returns the generated Ent migration schema used to
// invalidate cached test databases when the Ent schema changes.
func GeneratedMigrationSchema() string {
	return generatedMigrationSchema
}

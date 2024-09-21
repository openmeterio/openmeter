// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A very lightweigh migration tool to replace `ent.Schema.Create` calls.
package migrate

import (
	"embed"
	"io/fs"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const (
	MigrationsTable = "schema_om"
)

type Migrate = migrate.Migrate

//go:embed migrations
var OMMigrations embed.FS

// NewMigrate creates a new migrate instance.
func NewMigrate(conn string, fs fs.FS, fsPath string) (*Migrate, error) {
	d, err := iofs.New(fs, fsPath)
	if err != nil {
		return nil, err
	}
	return migrate.NewWithSourceInstance("iofs", d, conn)
}

func Up(conn string) error {
	conn, err := SetMigrationTableName(conn, MigrationsTable)
	if err != nil {
		return err
	}
	m, err := NewMigrate(conn, OMMigrations, "migrations")
	if err != nil {
		return err
	}

	defer m.Close()
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func SetMigrationTableName(conn, tableName string) (string, error) {
	parsedURL, err := url.Parse(conn)
	if err != nil {
		return "", err
	}

	values := parsedURL.Query()
	values.Set("x-migrations-table", tableName)
	parsedURL.RawQuery = values.Encode()

	return parsedURL.String(), nil
}

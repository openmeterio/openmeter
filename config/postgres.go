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

package config

import (
	"errors"

	"github.com/spf13/viper"
)

type PostgresConfig struct {
	// URL is the PostgreSQL database connection URL.
	URL string `yaml:"url"`
	// AutoMigrate is a flag that indicates whether the database should be automatically migrated.
	// Supported values are:
	// - "false" to disable auto-migration at startup
	// - "ent" to use ent Schema Upserts (the default value)
	// - "migration" to use the migrations directory
	AutoMigrate AutoMigrate `yaml:"autoMigrate"`
}

// Validate validates the configuration.
func (c PostgresConfig) Validate() error {
	if c.URL == "" {
		return errors.New("database URL is required")
	}
	if err := c.AutoMigrate.Validate(); err != nil {
		return err
	}

	return nil
}

func ConfigurePostgres(v *viper.Viper) {
	v.SetDefault("postgres.autoMigrate", "ent")
}

type AutoMigrate string

const (
	AutoMigrateEnt       AutoMigrate = "ent"
	AutoMigrateMigration AutoMigrate = "migration"
	AutoMigrateOff       AutoMigrate = "false"
)

func (a AutoMigrate) Enabled() bool {
	// For all other values it's enabled
	return !(a == "false")
}

func (a AutoMigrate) Validate() error {
	switch a {
	case AutoMigrateEnt, AutoMigrateMigration, AutoMigrateOff:
		return nil
	default:
		return errors.New("invalid auto-migrate value")
	}
}

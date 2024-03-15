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

package startup

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func DB(ctx context.Context, cfg config.PostgresConfig, db *db.Client) error {
	if !cfg.AutoMigrate.Enabled() {
		return nil
	}

	switch cfg.AutoMigrate {
	case config.AutoMigrateEnt:
		if err := db.Schema.Create(ctx); err != nil {
			return fmt.Errorf("failed to migrate db: %w", err)
		}
	case config.AutoMigrateMigration:
		if err := migrate.Up(cfg.URL); err != nil {
			return fmt.Errorf("failed to migrate db: %w", err)
		}
	}

	return nil
}

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

package credit

import (
	"log/slog"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/credit/postgres_connector"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/migrate"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func NewSchema(driver *sql.Driver) *migrate.Schema {
	return db.NewClient(db.Driver(driver)).Schema
}

func NewConnector(
	logger *slog.Logger,
	driver *sql.Driver,
	streamingConnector streaming.Connector,
	meterRepository meter.Repository,
) Connector {
	return postgres_connector.NewPostgresConnector(
		logger, db.NewClient(db.Driver(driver)), streamingConnector, meterRepository,
	)
}

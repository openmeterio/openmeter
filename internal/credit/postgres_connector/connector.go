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

package postgres_connector

import (
	"log/slog"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/streaming"
)

type PostgresConnector struct {
	logger             *slog.Logger
	db                 *db.Client
	streamingConnector streaming.Connector
	meterRepository    meter.Repository
}

// Implement the Connector interface
var _ credit.Connector = &PostgresConnector{}

func NewPostgresConnector(
	logger *slog.Logger,
	db *db.Client,
	streamingConnector streaming.Connector,
	meterRepository meter.Repository,
) credit.Connector {
	connector := PostgresConnector{
		logger:             logger,
		db:                 db,
		streamingConnector: streamingConnector,
		meterRepository:    meterRepository,
	}

	return &connector
}

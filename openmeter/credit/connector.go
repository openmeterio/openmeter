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
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type CreditConnector interface {
	BalanceConnector
	GrantConnector
}

type connector struct {
	// grants and balance snapshots are managed in this same package
	grantRepo           grant.Repo
	balanceSnapshotRepo balance.SnapshotRepo
	// external dependencies
	publisher          eventbus.Publisher
	ownerConnector     grant.OwnerConnector
	streamingConnector streaming.Connector
	logger             *slog.Logger
	// configuration
	snapshotGracePeriod time.Duration
	granularity         time.Duration
}

func NewCreditConnector(
	grantRepo grant.Repo,
	balanceSnapshotRepo balance.SnapshotRepo,
	ownerConnector grant.OwnerConnector,
	streamingConnector streaming.Connector,
	logger *slog.Logger,
	granularity time.Duration,
	publisher eventbus.Publisher,
) CreditConnector {
	return &connector{
		grantRepo:           grantRepo,
		balanceSnapshotRepo: balanceSnapshotRepo,
		ownerConnector:      ownerConnector,
		streamingConnector:  streamingConnector,
		logger:              logger,

		publisher: publisher,

		// TODO: make configurable
		granularity:         granularity,
		snapshotGracePeriod: time.Hour,
	}
}

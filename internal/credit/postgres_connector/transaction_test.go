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
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	meter_model "github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestTransaction(t *testing.T) {
	namespace := "default"
	meterRepository := meter_model.NewInMemoryRepository([]models.Meter{})

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector PostgresConnector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger)
	}{
		{
			name:        "Lock",
			description: "Should manage locks correctly",
			test: func(t *testing.T, connector PostgresConnector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()

				ledgerID := credit.NamespacedLedgerID{
					Namespace: namespace,
					ID:        ledger.ID,
				}
				// 1. Should succeed to obtain lock
				_, err := mutationTransaction(ctx, &connector, ledgerID, func(tx *db.Tx, ledgerEntity *db.Ledger) (*db.Ledger, error) {
					return ledgerEntity, nil
				})
				assert.NoError(t, err)

				// 2.1. Lock ledger
				tx, err := db_client.Tx(ctx)
				assert.NoError(t, err)
				_, err = lockLedger(tx, ctx, ledgerID)
				assert.NoError(t, err)

				var wg sync.WaitGroup
				var chMutationError chan error = make(chan error)
				var chCommitError chan error = make(chan error)
				wg.Add(2)

				// 2.2. Should wait until ledger is locked (waiting for 2.3. to unlock ledger)
				go func() {
					_, err := mutationTransaction(ctx, &connector, ledgerID, func(tx *db.Tx, ledgerEntity *db.Ledger) (*db.Ledger, error) {
						return ledgerEntity, nil
					})
					wg.Done()
					chMutationError <- err
				}()

				// 2.3. Unlock ledger (2.2. should proceed after this)
				go func() {
					err = tx.Commit()
					wg.Done()
					chCommitError <- err
				}()

				// Wait for 2.2. and 2.3. to finish
				wg.Wait()

				// Assert that 2.2. and 2.3. are successful
				mutationErr := <-chMutationError
				assert.NoError(t, mutationErr)

				commitErr := <-chCommitError
				assert.NoError(t, commitErr)

				// 3. Should succeed to obtain lock after commit
				_, err = mutationTransaction(ctx, &connector, ledgerID, func(tx *db.Tx, ledgerEntity *db.Ledger) (*db.Ledger, error) {
					return ledgerEntity, nil
				})
				assert.NoError(t, err)
			},
		},
		{
			name:        "LockWithCancel",
			description: "Should respect context cancel in locks",
			test: func(t *testing.T, connector PostgresConnector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				ledgerID := credit.NamespacedLedgerID{
					Namespace: namespace,
					ID:        ledger.ID,
				}
				// 1.1. Lock ledger
				// Limit the time to wait to obtain the lock
				ctxLock, cancelLock := context.WithCancel(ctx)

				tx, err := db_client.Tx(ctxLock)
				assert.NoError(t, err)
				_, err = lockLedger(tx, ctxLock, ledgerID)
				assert.NoError(t, err)

				// 1.2. Lock will timeout
				cancelLock()

				// 2.2. Should wait until ledger is locked (waiting for 1.2. to unlock ledger)
				_, err = mutationTransaction(ctx, &connector, ledgerID, func(tx *db.Tx, ledgerEntity *db.Ledger) (*db.Ledger, error) {
					return ledgerEntity, nil
				})
				assert.NoError(t, err)
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tc.description)
			driver := initDB(t)
			databaseClient := db.NewClient(db.Driver(driver))
			defer databaseClient.Close()

			// Note: lock manager cannot be shared between tests as these parallel tests write the same ledger
			streamingConnector := newMockStreamingConnector()
			connector := PostgresConnector{
				logger:             slog.Default(),
				db:                 databaseClient,
				streamingConnector: streamingConnector,
				meterRepository:    meterRepository,
			}
			// let's provision a ledger
			ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
				Namespace: namespace,
				Subject:   ulid.Make().String(),
			})

			assert.NoError(t, err)

			tc.test(t, connector, streamingConnector, databaseClient, ledger)
		})
	}
}

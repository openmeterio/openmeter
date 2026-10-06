package feature_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Only the connector's transaction boundary is under test; the outbox owns event persistence tests.
type transactionCheckingPublisher struct {
	eventbus.Publisher
	Driver transaction.Driver
	Err    error
}

func (p *transactionCheckingPublisher) Publish(ctx context.Context, _ marshaler.Event) error {
	driver, err := transaction.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("publisher requires the feature transaction: %w", err)
	}
	p.Driver = driver
	return p.Err
}

func TestFeatureConnectorTransactions(t *testing.T) {
	operations := []struct {
		Name string
		Run  func(context.Context, feature.FeatureConnector, feature.Feature) error
	}{
		{
			Name: "create",
			Run: func(ctx context.Context, connector feature.FeatureConnector, seeded feature.Feature) error {
				_, err := connector.CreateFeature(ctx, feature.CreateFeatureInputs{
					Namespace: seeded.Namespace,
					Name:      "Created feature",
					Key:       "created",
				})
				return err
			},
		},
		{
			Name: "update",
			Run: func(ctx context.Context, connector feature.FeatureConnector, seeded feature.Feature) error {
				_, err := connector.UpdateFeature(ctx, feature.UpdateFeatureInputs{
					Namespace: seeded.Namespace,
					ID:        seeded.ID,
					UnitCost: nullable.NewNullableWithValue(feature.UnitCost{
						Type:   feature.UnitCostTypeManual,
						Manual: &feature.ManualUnitCost{Amount: alpacadecimal.NewFromInt(2)},
					}),
				})
				return err
			},
		},
		{
			Name: "archive",
			Run: func(ctx context.Context, connector feature.FeatureConnector, seeded feature.Feature) error {
				return connector.ArchiveFeature(ctx, models.NamespacedID{Namespace: seeded.Namespace, ID: seeded.ID})
			},
		},
	}
	for _, operation := range operations {
		t.Run(operation.Name, func(t *testing.T) {
			for _, outcome := range []string{"publisher failure", "outer rollback"} {
				t.Run(outcome, func(t *testing.T) {
					// given a real feature repository and a publisher that requires a transaction context
					testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
					t.Cleanup(func() { testDB.Close(t) })
					repo := adapter.NewPostgresFeatureRepo(testDB.EntDriver.Client(), testutils.NewDiscardLogger(t))
					seeded, err := repo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
						Namespace: "feature-transactions",
						Name:      "Original feature",
						Key:       "original",
					})
					require.NoError(t, err)
					publisherErr := errors.New("publisher failed")
					publisher := &transactionCheckingPublisher{Publisher: eventbus.NewMock(t)}
					if outcome == "publisher failure" {
						publisher.Err = publisherErr
					}
					connector := feature.NewFeatureConnector(repo, nil, publisher)
					ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
					defer cancel()
					rollbackErr := errors.New("outer operation failed")
					var outerDriver transaction.Driver

					// when publication fails or the caller rolls back
					if outcome == "outer rollback" {
						err = transaction.RunWithNoValue(ctx, repo, func(ctx context.Context) error {
							var err error
							outerDriver, err = transaction.GetDriverFromContext(ctx)
							if err != nil {
								return err
							}
							if err := operation.Run(ctx, connector, seeded); err != nil {
								return err
							}
							return rollbackErr
						})
						require.ErrorIs(t, err, rollbackErr)
						require.Same(t, outerDriver, publisher.Driver)
					} else {
						err = operation.Run(ctx, connector, seeded)
						require.ErrorIs(t, err, publisherErr)
					}

					// then the transaction reaches publication and none of its feature changes persist
					require.NotNil(t, publisher.Driver)
					if operation.Name == "create" {
						_, err := repo.GetByIdOrKey(t.Context(), seeded.Namespace, "created", bool(feature.IncludeArchivedFeatureTrue))
						require.ErrorAs(t, err, new(*feature.FeatureNotFoundError))
					} else {
						stored, err := repo.GetByIdOrKey(t.Context(), seeded.Namespace, seeded.ID, bool(feature.IncludeArchivedFeatureTrue))
						require.NoError(t, err)
						require.Nil(t, stored.UnitCost)
						require.Nil(t, stored.ArchivedAt)
					}
				})
			}
		})
	}
}

package entitlement_test

import (
	"context"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	credit_postgres_adapter "github.com/openmeterio/openmeter/internal/credit/postgres_adapter"
	credit_postgres_adapter_db "github.com/openmeterio/openmeter/internal/credit/postgres_adapter/ent/db"
	"github.com/openmeterio/openmeter/internal/entitlement"
	feature_postgres_adapter "github.com/openmeterio/openmeter/internal/productcatalog/postgres_adapter"
	feature_postgres_adapter_db "github.com/openmeterio/openmeter/internal/productcatalog/postgres_adapter/ent/db"
	streaming_testutils "github.com/openmeterio/openmeter/internal/streaming/testutils"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/stretchr/testify/assert"
)

type mockEDBAdapter struct {
	entitlements map[entitlement.NamespacedEntitlementID]entitlement.Entitlement
}

var _ entitlement.EntitlementDBConnector = (*mockEDBAdapter)(nil)

func (m *mockEDBAdapter) GetEntitlement(ctx context.Context, id entitlement.NamespacedEntitlementID) (*entitlement.Entitlement, error) {
	entitl, ok := m.entitlements[id]
	if !ok {
		return nil, &entitlement.EntitlementNotFoundError{EntitlementID: id}
	}
	return &entitl, nil
}

func TestE2E(t *testing.T) {
	t1, err := time.Parse(time.RFC3339, "2024-03-01T00:00:00Z")
	assert.NoError(t, err)

	streaming := streaming_testutils.NewMockStreamingConnector(t, streaming_testutils.MockStreamingConnectorParams{
		DefaultHighwatermark: t1.AddDate(-1, 0, 0), //
	})
	driver := testutils.InitPostgresDB(t)
	featureDBClient := feature_postgres_adapter_db.NewClient(feature_postgres_adapter_db.Driver(driver))
	if err := featureDBClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	featureDB := feature_postgres_adapter.NewPostgresFeatureDBAdapter(featureDBClient, testutils.NewLogger(t))
	owner := entitlement.NewEntitlementGrantOwnerAdapter(
		featureDB,
		&mockEDBAdapter{
			entitlements: map[entitlement.NamespacedEntitlementID]entitlement.Entitlement{
				{Namespace: "ns1", ID: "ent1"}: {
					ID: "ent1",
				},
			},
		},
		testutils.NewLogger(t),
	)

	grantDbClient := credit_postgres_adapter_db.NewClient(credit_postgres_adapter_db.Driver(driver))
	if err := grantDbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}
	grantDbConn := credit_postgres_adapter.NewPostgresGrantDBAdapter(grantDbClient)

	balanceSnapshotDbConn := credit_postgres_adapter.NewPostgresBalanceSnapshotDBAdapter(grantDbClient)

	balance := credit.NewBalanceConnector(
		grantDbConn,
		balanceSnapshotDbConn,
		owner,
		streaming,
		testutils.NewLogger(t),
	)

	connector := entitlement.NewEntitlementBalanceConnector(
		streaming,
		owner,
		balance,
	)

	queryTime := t1.AddDate(0, 0, 1)

	entBalance, err := connector.GetEntitlementBalance(context.Background(), entitlement.NamespacedEntitlementID{Namespace: "ns1", ID: "ent1"}, queryTime)

	assert.NoError(t, err)
	assert.NotNil(t, entBalance)

}

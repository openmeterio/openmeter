package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	credit_connector "github.com/openmeterio/openmeter/internal/credit"
	inmemory_lock "github.com/openmeterio/openmeter/internal/credit/inmemory_lock"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	meter_internal "github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
	product_model "github.com/openmeterio/openmeter/pkg/product"
)

func TestProduct(t *testing.T) {
	namespace := "default"
	meter := models.Meter{
		Namespace: namespace,
		ID:        "meter-1",
		Slug:      "meter-1",
		GroupBy:   map[string]string{"key": "$.path"},
	}
	meters := []models.Meter{meter}
	meterRepository := meter_internal.NewInMemoryRepository(meters)
	lockManager := inmemory_lock.NewLockManager(time.Second * 10)

	testProduct := product_model.Product{
		Namespace: namespace,
		Name:      "product-1",
		MeterSlug: meter.Slug,
		MeterGroupByFilters: &map[string]string{
			"key": "value",
		},
	}

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, db_client *db.Client)
	}{
		{
			name:        "CreateProduct",
			description: "Create a product in the database",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				ctx := context.Background()
				productIn := testProduct
				productOut, err := connector.CreateProduct(ctx, namespace, productIn)
				assert.NoError(t, err)
				// assert count
				assert.Equal(t, 1, db_client.Product.Query().CountX(ctx))
				// assert fields
				assert.NotNil(t, productOut.ID)
				productIn.ID = productOut.ID
				assert.Equal(t, productIn, productOut)
			},
		},
		{
			name:        "GetProduct",
			description: "Get a product in the database",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				ctx := context.Background()
				productIn, err := connector.CreateProduct(ctx, namespace, testProduct)
				assert.NoError(t, err)

				productOut, err := connector.GetProduct(ctx, namespace, *productIn.ID)
				assert.NoError(t, err)
				assert.Equal(t, productIn, productOut)
			},
		},
		{
			name:        "DeleteProduct",
			description: "Delete a product in the database",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				ctx := context.Background()
				p, err := connector.CreateProduct(ctx, namespace, testProduct)
				assert.NoError(t, err)

				err = connector.DeleteProduct(ctx, namespace, *p.ID)
				assert.NoError(t, err)

				// assert
				p, err = connector.GetProduct(ctx, namespace, *p.ID)
				assert.NoError(t, err)
				assert.True(t, p.Archived)
			},
		},
		{
			name:        "ListProducts",
			description: "List products in the database",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				ctx := context.Background()
				product, err := connector.CreateProduct(ctx, namespace, testProduct)
				assert.NoError(t, err)

				products, err := connector.ListProducts(ctx, namespace, credit_connector.ListProductsParams{})
				assert.NoError(t, err)
				assert.Len(t, products, 1)
				assert.Equal(t, product, products[0])
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
			connector := NewPostgresConnector(slog.Default(), databaseClient, nil, meterRepository, lockManager)
			tc.test(t, connector, databaseClient)
		})
	}
}

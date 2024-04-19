package postgres_connector

import (
	"context"
	"fmt"

	connector "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_product "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/product"
	product_model "github.com/openmeterio/openmeter/pkg/product"
)

// CreateProduct creates a product.
func (c *PostgresConnector) CreateProduct(ctx context.Context, namespace string, productIn product_model.Product) (product_model.Product, error) {
	query := c.db.Product.Create().
		SetNillableID(productIn.ID).
		SetName(productIn.Name).
		SetNamespace(namespace).
		SetMeterSlug(productIn.MeterSlug)

	if productIn.MeterGroupByFilters != nil {
		query.SetMeterGroupByFilters(*productIn.MeterGroupByFilters)
	}

	entity, err := query.Save(ctx)
	if err != nil {
		return product_model.Product{}, fmt.Errorf("failed to create product: %w", err)
	}

	productOut := mapProductEntity(entity)
	return productOut, nil
}

// DeleteProduct deletes a product.
func (c *PostgresConnector) DeleteProduct(ctx context.Context, namespace string, id string) error {
	err := c.db.Product.UpdateOneID(id).SetArchived(true).Exec(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return &product_model.ProductNotFoundError{ID: id}
		}

		return fmt.Errorf("failed to delete product: %w", err)
	}

	return nil
}

// ListProducts lists products.
func (c *PostgresConnector) ListProducts(ctx context.Context, namespace string, params connector.ListProductsParams) ([]product_model.Product, error) {
	query := c.db.Product.Query()
	if !params.IncludeArchived {
		query = query.Where(db_product.ArchivedEQ(false))
	}

	entities, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	var list []product_model.Product
	for _, entity := range entities {
		product := mapProductEntity(entity)
		list = append(list, product)
	}

	return list, nil
}

// GetProduct gets a single product by ID.
func (c *PostgresConnector) GetProduct(ctx context.Context, namespace string, id string) (product_model.Product, error) {
	entity, err := c.db.Product.Get(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			return product_model.Product{}, &product_model.ProductNotFoundError{ID: id}
		}

		return product_model.Product{}, fmt.Errorf("failed to get product: %w", err)
	}

	productOut := mapProductEntity(entity)
	return productOut, nil
}

// mapProductEntity maps a database product entity to a product model.
func mapProductEntity(entity *db.Product) product_model.Product {
	product := product_model.Product{
		ID:        &entity.ID,
		Namespace: entity.Namespace,
		Name:      entity.Name,
		MeterSlug: entity.MeterSlug,
		Archived:  entity.Archived,
	}

	if len(entity.MeterGroupByFilters) > 0 {
		product.MeterGroupByFilters = &entity.MeterGroupByFilters
	}

	return product
}

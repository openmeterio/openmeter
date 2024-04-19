package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/api"
	connector "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
	product_model "github.com/openmeterio/openmeter/pkg/product"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// Get product, GET:/api/v1/products/{productId}
func (a *Router) GetProduct(w http.ResponseWriter, r *http.Request, productId api.ProductId) {
	ctx := contextx.WithAttr(r.Context(), "operation", "GetProduct")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	product, err := a.config.CreditConnector.GetProduct(ctx, namespace, productId)
	if err != nil {
		if _, ok := err.(*product_model.ProductNotFoundError); ok {
			models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w, r)
			return
		}

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	_ = render.Render(w, r, product)
}

// List products: GET /api/v1/products
func (a *Router) ListProducts(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "ListProducts")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	products, err := a.config.CreditConnector.ListProducts(ctx, namespace, connector.ListProductsParams{})
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}
	list := slicesx.Map[product_model.Product, render.Renderer](products, func(product product_model.Product) render.Renderer {
		return &product
	})

	_ = render.RenderList(w, r, list)
}

// Create product, POST: /api/v1/products
func (a *Router) CreateProduct(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "CreateProduct")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Parse request body
	productIn := &api.CreateProductJSONRequestBody{}
	if err := render.DecodeJSON(r.Body, productIn); err != nil {
		err := fmt.Errorf("decode json: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)

		return
	}

	id := ulid.Make().String()
	productIn.ID = &id

	productOut, err := a.config.CreditConnector.CreateProduct(ctx, namespace, *productIn)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	render.Status(r, http.StatusCreated)
	_ = render.Render(w, r, productOut)
}

// Delete product, DELETE:/api/v1/products/{productId}
func (a *Router) DeleteProduct(w http.ResponseWriter, r *http.Request, productId api.ProductId) {
	ctx := contextx.WithAttr(r.Context(), "operation", "DeleteProduct")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	_, err := a.config.CreditConnector.GetProduct(ctx, namespace, productId)
	if err != nil {
		if _, ok := err.(*product_model.ProductNotFoundError); ok {
			models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w, r)
			return
		}

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	err = a.config.CreditConnector.DeleteProduct(ctx, namespace, productId)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

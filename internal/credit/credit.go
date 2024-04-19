package credit

import (
	"context"
	"time"

	credit_model "github.com/openmeterio/openmeter/pkg/credit"
	product_model "github.com/openmeterio/openmeter/pkg/product"
)

type ListGrantsParams struct {
	Subjects          []string
	From              *time.Time
	To                *time.Time
	FromHighWatermark bool
	IncludeVoid       bool
}

type ListProductsParams struct {
	IncludeArchived bool
}

type Connector interface {
	// Grant
	CreateGrant(ctx context.Context, namespace string, grant credit_model.Grant) (credit_model.Grant, error)
	VoidGrant(ctx context.Context, namespace string, grant credit_model.Grant) (credit_model.Grant, error)
	ListGrants(ctx context.Context, namespace string, params ListGrantsParams) ([]credit_model.Grant, error)
	GetGrant(ctx context.Context, namespace string, id string) (credit_model.Grant, error)

	// Credit
	GetBalance(ctx context.Context, namespace string, subject string, cutline time.Time) (credit_model.Balance, error)
	GetLedger(ctx context.Context, namespace string, subject string, from time.Time, to time.Time) (credit_model.LedgerEntryList, error)
	GetHighWatermark(ctx context.Context, namespace string, subject string) (credit_model.HighWatermark, error)
	Reset(ctx context.Context, namespace string, reset credit_model.Reset) (credit_model.Reset, []credit_model.Grant, error)

	// Product
	CreateProduct(ctx context.Context, namespace string, product product_model.Product) (product_model.Product, error)
	DeleteProduct(ctx context.Context, namespace string, id string) error
	ListProducts(ctx context.Context, namespace string, params ListProductsParams) ([]product_model.Product, error)
	GetProduct(ctx context.Context, namespace string, id string) (product_model.Product, error)
}

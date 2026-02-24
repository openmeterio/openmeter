package taxcode

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Repository interface {
	entutils.TxCreator

	CreateTaxCode(ctx context.Context, input CreateTaxCodeInput) (TaxCode, error)
	UpdateTaxCode(ctx context.Context, input UpdateTaxCodeInput) (TaxCode, error)
	ListTaxCodes(ctx context.Context, input ListTaxCodesInput) (pagination.Result[TaxCode], error)
	GetTaxCode(ctx context.Context, input GetTaxCodeInput) (TaxCode, error)
	DeleteTaxCode(ctx context.Context, input DeleteTaxCodeInput) error
}

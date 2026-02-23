package service

import (
	"context"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type service struct {
	adapter taxcode.Repository
	logger  *slog.Logger
}

func New(adapter taxcode.Repository, logger *slog.Logger) taxcode.Service {
	return &service{
		adapter: adapter,
		logger:  logger,
	}
}

func (s *service) CreateTaxCode(ctx context.Context, input taxcode.CreateTaxCodeInput) (taxcode.TaxCode, error) {
	return s.adapter.CreateTaxCode(ctx, input)
}

func (s *service) UpdateTaxCode(ctx context.Context, input taxcode.UpdateTaxCodeInput) (taxcode.TaxCode, error) {
	return s.adapter.UpdateTaxCode(ctx, input)
}

func (s *service) ListTaxCodes(ctx context.Context, input taxcode.ListTaxCodesInput) (pagination.Result[taxcode.TaxCode], error) {
	return s.adapter.ListTaxCodes(ctx, input)
}

func (s *service) GetTaxCode(ctx context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
	return s.adapter.GetTaxCode(ctx, input)
}

func (s *service) DeleteTaxCode(ctx context.Context, input taxcode.DeleteTaxCodeInput) error {
	return s.adapter.DeleteTaxCode(ctx, input)
}

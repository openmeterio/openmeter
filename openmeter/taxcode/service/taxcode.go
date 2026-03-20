package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
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
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		return s.adapter.CreateTaxCode(ctx, input)
	})
}

func (s *service) UpdateTaxCode(ctx context.Context, input taxcode.UpdateTaxCodeInput) (taxcode.TaxCode, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		return s.adapter.UpdateTaxCode(ctx, input)
	})
}

func (s *service) ListTaxCodes(ctx context.Context, input taxcode.ListTaxCodesInput) (pagination.Result[taxcode.TaxCode], error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[taxcode.TaxCode], error) {
		return s.adapter.ListTaxCodes(ctx, input)
	})
}

func (s *service) GetTaxCode(ctx context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		return s.adapter.GetTaxCode(ctx, input)
	})
}

func (s *service) GetTaxCodeByAppMapping(ctx context.Context, input taxcode.GetTaxCodeByAppMappingInput) (taxcode.TaxCode, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		return s.adapter.GetTaxCodeByAppMapping(ctx, input)
	})
}

// GetOrCreateByAppMapping looks up a TaxCode by its app mapping. If none exists,
// it creates one with a key derived from the app-specific code.
func (s *service) GetOrCreateByAppMapping(ctx context.Context, input taxcode.GetOrCreateByAppMappingInput) (taxcode.TaxCode, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		// Try to find an existing TaxCode with this app mapping.
		tc, err := s.adapter.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
			Namespace: input.Namespace,
			AppType:   input.AppType,
			TaxCode:   input.TaxCode,
		})
		if err == nil {
			return tc, nil
		}

		// Not found — create a new TaxCode.
		key := fmt.Sprintf("%s_%s", input.AppType, input.TaxCode)

		tc, err = s.adapter.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
			Namespace: input.Namespace,
			Key:       key,
			Name:      input.TaxCode,
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: input.AppType, TaxCode: input.TaxCode},
			},
		})
		if err != nil {
			// Another request may have created it concurrently.
			if models.IsGenericConflictError(err) {
				return s.adapter.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
					Namespace: input.Namespace,
					AppType:   input.AppType,
					TaxCode:   input.TaxCode,
				})
			}

			return taxcode.TaxCode{}, err
		}

		return tc, nil
	})
}

func (s *service) DeleteTaxCode(ctx context.Context, input taxcode.DeleteTaxCodeInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.DeleteTaxCode(ctx, input)
	})
}

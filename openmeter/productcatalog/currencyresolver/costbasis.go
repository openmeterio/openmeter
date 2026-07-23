package currencyresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type costBasisChecker struct {
	service currencies.CostBasisService
}

func NewCostBasisChecker(service currencies.CostBasisService) (productcatalog.CostBasisChecker, error) {
	if service == nil {
		return nil, errors.New("cost basis service is required")
	}

	return &costBasisChecker{service: service}, nil
}

func (c *costBasisChecker) HasCostBasis(ctx context.Context, namespace, customCurrencyID string, fiatCurrencyCode currencyx.Code) (bool, error) {
	if namespace == "" {
		return false, errors.New("namespace is required")
	}

	if customCurrencyID == "" {
		return false, errors.New("custom currency ID is required")
	}

	if err := fiatCurrencyCode.Validate(); err != nil {
		return false, fmt.Errorf("valid fiat currency code is required: %w", err)
	}

	if !fiatCurrencyCode.IsFiat() {
		return false, errors.New("valid fiat currency code is required")
	}

	result, err := c.service.ListCostBases(ctx, currencies.ListCostBasesInput{
		Page:           pagination.NewPage(1, 1),
		Namespace:      namespace,
		CurrencyID:     customCurrencyID,
		FilterFiatCode: &fiatCurrencyCode,
	})
	if err != nil {
		return false, fmt.Errorf("listing cost bases: %w", err)
	}

	return len(result.Items) > 0, nil
}

var _ productcatalog.CostBasisChecker = (*costBasisChecker)(nil)

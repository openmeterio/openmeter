package currencies

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*CurrencyRef)(nil)

// CurrencyRef identifies a currency by its managed resource ID or code.
// ID takes precedence when both fields are set.
type CurrencyRef struct {
	ID   string         `json:"id,omitempty"`
	Code currencyx.Code `json:"code,omitempty"`
}

func (r CurrencyRef) Validate() error {
	if r.ID != "" {
		return nil
	}

	if r.Code == "" {
		return errors.New("currency id or code is required")
	}

	return nil
}

type CurrencyResolver interface {
	ResolveCurrency(ctx context.Context, namespace string, ref CurrencyRef) (*Currency, error)
	BatchResolveCurrencies(ctx context.Context, namespace string, refs ...CurrencyRef) (map[CurrencyRef]*Currency, error)
	WithNamespace(namespace string) NamespacedCurrencyResolver
}

type NamespacedCurrencyResolver interface {
	ResolveCurrency(ctx context.Context, ref CurrencyRef) (*Currency, error)
	BatchResolveCurrencies(ctx context.Context, refs ...CurrencyRef) (map[CurrencyRef]*Currency, error)
	Namespace() string
}

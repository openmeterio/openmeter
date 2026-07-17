package subscription

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CostBasisMode string

const (
	CostBasisModeDynamic CostBasisMode = "dynamic"
	CostBasisModePinned  CostBasisMode = "pinned"
)

func (CostBasisMode) Values() []string {
	return []string{
		string(CostBasisModeDynamic),
		string(CostBasisModePinned),
	}
}

func (m CostBasisMode) OrDefault() CostBasisMode {
	if m == "" {
		return CostBasisModeDynamic
	}

	return m
}

func (m CostBasisMode) Validate() error {
	switch m.OrDefault() {
	case CostBasisModeDynamic, CostBasisModePinned:
		return nil
	default:
		return fmt.Errorf("invalid cost basis mode %q", m)
	}
}

func (m CostBasisMode) IsPinned() bool {
	return m.OrDefault() == CostBasisModePinned
}

type CostBasisPin struct {
	models.NamespacedID
	models.ManagedModel

	CustomCurrencyID string               `json:"customCurrencyId"`
	InvoiceCurrency  currencyx.Code       `json:"invoiceCurrency"`
	CostBasis        currencies.CostBasis `json:"costBasis"`
}

func (p CostBasisPin) Validate() error {
	var errs []error

	if err := p.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if p.CustomCurrencyID == "" {
		errs = append(errs, errors.New("custom currency ID is required"))
	}

	if err := p.InvoiceCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invoice currency: %w", err))
	} else if !p.InvoiceCurrency.IsFiat() {
		errs = append(errs, errors.New("invoice currency must be fiat"))
	}

	if p.CostBasis.ID == "" {
		errs = append(errs, errors.New("cost basis is required"))
	} else {
		if p.CostBasis.CurrencyID != p.CustomCurrencyID {
			errs = append(errs, errors.New("cost basis custom currency does not match pin"))
		}
		if currencyx.Code(p.CostBasis.FiatCode) != p.InvoiceCurrency {
			errs = append(errs, errors.New("cost basis fiat currency does not match pin"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s SubscriptionSpec) HasCustomCurrencyBillables() bool {
	for _, phase := range s.Phases {
		if phase == nil {
			continue
		}

		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				if item == nil || item.RateCard == nil {
					continue
				}

				meta := item.RateCard.AsMeta()
				if meta.Price != nil && meta.Currency != nil && meta.Currency.IsCustom() {
					return true
				}
			}
		}
	}

	return false
}

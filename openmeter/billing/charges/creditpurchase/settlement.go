package creditpurchase

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SettlementType string

const (
	SettlementTypeInvoice     SettlementType = "invoice"
	SettlementTypeExternal    SettlementType = "external"
	SettlementTypePromotional SettlementType = "promotional"
)

func (s SettlementType) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid credit purchase settlement type: %s", s))
	}
	return nil
}

func (s SettlementType) Values() []string {
	return []string{
		string(SettlementTypeInvoice),
		string(SettlementTypeExternal),
		string(SettlementTypePromotional),
	}
}

type InitialPaymentSettlementStatus string

const (
	CreatedInitialPaymentSettlementStatus    InitialPaymentSettlementStatus = "created"
	AuthorizedInitialPaymentSettlementStatus InitialPaymentSettlementStatus = "authorized"
	SettledInitialPaymentSettlementStatus    InitialPaymentSettlementStatus = "settled"
)

func (s InitialPaymentSettlementStatus) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid payment settlement status: %s", s))
	}
	return nil
}

func (s InitialPaymentSettlementStatus) Values() []string {
	return []string{
		string(CreatedInitialPaymentSettlementStatus),
		string(AuthorizedInitialPaymentSettlementStatus),
		string(SettledInitialPaymentSettlementStatus),
	}
}

func (s InitialPaymentSettlementStatus) In(statuses ...InitialPaymentSettlementStatus) bool {
	return slices.Contains(statuses, s)
}

type ExternalSettlement struct {
	InitialStatus InitialPaymentSettlementStatus `json:"status"`
}

func (s ExternalSettlement) Validate() error {
	var errs []error

	if err := s.InitialStatus.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("initial status: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type PromotionalSettlement struct{}

func (s PromotionalSettlement) Validate() error {
	return nil
}

type Settlement struct {
	t SettlementType

	external    *ExternalSettlement
	promotional *PromotionalSettlement
}

func (s Settlement) MarshalJSON() ([]byte, error) {
	var serde interface{}

	switch s.t {
	case SettlementTypeInvoice:
		serde = struct {
			Type SettlementType `json:"type"`
		}{Type: SettlementTypeInvoice}
	case SettlementTypeExternal:
		if s.external == nil {
			return nil, fmt.Errorf("settlement: external is nil")
		}

		serde = struct {
			Type SettlementType `json:"type"`
			*ExternalSettlement
		}{
			Type:               SettlementTypeExternal,
			ExternalSettlement: s.external,
		}
	case SettlementTypePromotional:
		serde = struct {
			Type SettlementType `json:"type"`
		}{
			Type: SettlementTypePromotional,
		}
	default:
		return nil, fmt.Errorf("invalid credit purchase settlement type: %s", s.t)
	}

	b, err := json.Marshal(serde)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON serialize CreditPurchaseSettlement: %w", err)
	}

	return b, nil
}

func (s *Settlement) UnmarshalJSON(bytes []byte) error {
	serde := &struct {
		Type SettlementType `json:"type"`
	}{}

	if err := json.Unmarshal(bytes, serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize CreditPurchaseSettlement type: %w", err)
	}

	switch serde.Type {
	case SettlementTypeInvoice:
		s.t = SettlementTypeInvoice
	case SettlementTypeExternal:
		v := &ExternalSettlement{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize ExternalCreditPurchaseSettlement: %w", err)
		}

		s.external = v
		s.t = SettlementTypeExternal
	case SettlementTypePromotional:
		s.promotional = &PromotionalSettlement{}
		s.t = SettlementTypePromotional
	default:
		return fmt.Errorf("invalid credit purchase settlement type: %s", serde.Type)
	}

	return nil
}

func NewInvoiceSettlement() Settlement {
	return Settlement{t: SettlementTypeInvoice}
}

func NewSettlement[T ExternalSettlement | PromotionalSettlement](settlement T) Settlement {
	switch v := any(settlement).(type) {
	case ExternalSettlement:
		return Settlement{
			t:        SettlementTypeExternal,
			external: &v,
		}
	case PromotionalSettlement:
		return Settlement{
			t:           SettlementTypePromotional,
			promotional: &v,
		}
	default:
		return Settlement{}
	}
}

func (s Settlement) Type() SettlementType {
	return s.t
}

func (s Settlement) Validate() error {
	switch s.t {
	case SettlementTypeInvoice:
	case SettlementTypeExternal:
		if s.external == nil {
			return models.NewGenericValidationError(fmt.Errorf("external is required"))
		}

		if err := s.external.Validate(); err != nil {
			return models.NewGenericValidationError(fmt.Errorf("external: %w", err))
		}
	case SettlementTypePromotional:
		if s.promotional == nil {
			return models.NewGenericValidationError(fmt.Errorf("promotional is required"))
		}

		if err := s.promotional.Validate(); err != nil {
			return models.NewGenericValidationError(fmt.Errorf("promotional: %w", err))
		}
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid credit purchase settlement type: %s", s.t))
	}
	return nil
}

func (s Settlement) AsExternalSettlement() (ExternalSettlement, error) {
	if s.t != SettlementTypeExternal {
		return ExternalSettlement{}, fmt.Errorf("settlement is not an external settlement")
	}

	if s.external == nil {
		return ExternalSettlement{}, fmt.Errorf("external is nil")
	}

	return *s.external, nil
}

func (s Settlement) AsPromotionalSettlement() (PromotionalSettlement, error) {
	if s.t != SettlementTypePromotional {
		return PromotionalSettlement{}, fmt.Errorf("settlement is not a promotional settlement")
	}

	if s.promotional == nil {
		return PromotionalSettlement{}, fmt.Errorf("promotional is nil")
	}

	return *s.promotional, nil
}

// PersistedSettlement is the temporary compatibility representation stored in
// charge_credit_purchases.settlement. Currency and cost basis remain here only
// so old application binaries can read rows written during the schema-level
// rollout. Remove them when the schema-level transition has concluded.
type PersistedSettlement struct {
	Type          SettlementType                  `json:"type"`
	Currency      *currencyx.FiatCode             `json:"currency,omitempty"`
	CostBasis     *alpacadecimal.Decimal          `json:"costBasis,omitempty"`
	InitialStatus *InitialPaymentSettlementStatus `json:"status,omitempty"`
}

func (s PersistedSettlement) Validate() error {
	var errs []error

	if err := s.Type.Validate(); err != nil {
		errs = append(errs, err)
	}

	switch s.Type {
	case SettlementTypeInvoice, SettlementTypeExternal:
		if s.Currency == nil {
			errs = append(errs, errors.New("currency is required"))
		} else if err := s.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}

		if s.CostBasis == nil {
			errs = append(errs, errors.New("cost basis is required"))
		} else if !s.CostBasis.IsPositive() {
			errs = append(errs, errors.New("cost basis must be positive"))
		}
	case SettlementTypePromotional:
		if s.Currency != nil || s.CostBasis != nil || s.InitialStatus != nil {
			errs = append(errs, errors.New("promotional settlement cannot contain payment compatibility fields"))
		}
	}

	if s.Type == SettlementTypeExternal {
		if s.InitialStatus == nil {
			errs = append(errs, errors.New("initial status is required"))
		} else if err := s.InitialStatus.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("initial status: %w", err))
		}
	} else if s.InitialStatus != nil {
		errs = append(errs, errors.New("initial status is only valid for external settlement"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func NewPersistedSettlement(settlement Settlement, resolvedCostBasis *ResolvedCostBasis) (PersistedSettlement, error) {
	if err := settlement.Validate(); err != nil {
		return PersistedSettlement{}, err
	}

	persisted := PersistedSettlement{Type: settlement.Type()}

	switch settlement.Type() {
	case SettlementTypeInvoice, SettlementTypeExternal:
		if resolvedCostBasis == nil {
			return PersistedSettlement{}, errors.New("resolved cost basis is required for payment-backed settlement")
		}
		if err := resolvedCostBasis.Validate(); err != nil {
			return PersistedSettlement{}, fmt.Errorf("resolved cost basis: %w", err)
		}

		currency := resolvedCostBasis.FiatCurrency.GetFiatCode()
		rate := resolvedCostBasis.Rate
		persisted.Currency = &currency
		persisted.CostBasis = &rate
	case SettlementTypePromotional:
	}

	if settlement.Type() == SettlementTypeExternal {
		external, err := settlement.AsExternalSettlement()
		if err != nil {
			return PersistedSettlement{}, err
		}

		persisted.InitialStatus = &external.InitialStatus
	}

	return persisted, persisted.Validate()
}

func (s PersistedSettlement) AsSettlement() (Settlement, error) {
	if err := s.Validate(); err != nil {
		return Settlement{}, err
	}

	switch s.Type {
	case SettlementTypeInvoice:
		return NewInvoiceSettlement(), nil
	case SettlementTypeExternal:
		return NewSettlement(ExternalSettlement{InitialStatus: *s.InitialStatus}), nil
	case SettlementTypePromotional:
		return NewSettlement(PromotionalSettlement{}), nil
	default:
		return Settlement{}, fmt.Errorf("invalid credit purchase settlement type: %s", s.Type)
	}
}

func (s PersistedSettlement) GetCostBasis() (alpacadecimal.Decimal, error) {
	if err := s.Validate(); err != nil {
		return alpacadecimal.Zero, err
	}

	if s.Type == SettlementTypePromotional {
		return alpacadecimal.Zero, nil
	}

	return *s.CostBasis, nil
}

func (s PersistedSettlement) GetCurrency() (*currencyx.FiatCode, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	return s.Currency, nil
}

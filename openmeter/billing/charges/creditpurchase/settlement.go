package creditpurchase

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

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

package creditpurchase

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type SettlementType string

const (
	SettlementTypeInvoice     SettlementType = "invoice"
	SettlementTypeExternal    SettlementType = "external"
	SettlementTypePromotional SettlementType = "promotional"
)

func (s SettlementType) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid credit purchase settlement type: %s", s)
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

type GenericSettlement struct {
	Currency  currencyx.Code            `json:"currency"`
	CostBasis alpacadecimal.Decimal     `json:"costBasis"`
	TaxConfig *productcatalog.TaxConfig `json:"taxConfig"`
}

func (s GenericSettlement) Validate() error {
	var errs []error

	if err := s.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement currency: %w", err))
	}

	if s.CostBasis.IsNegative() {
		errs = append(errs, fmt.Errorf("cost basis must be zero or positive"))
	}

	if s.TaxConfig != nil {
		if err := s.TaxConfig.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax config: %w", err))
		}
	}

	return errors.Join(errs...)
}

type InvoiceSettlement struct {
	GenericSettlement
}

func (s InvoiceSettlement) Validate() error {
	var errs []error

	if err := s.GenericSettlement.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("generic settlement: %w", err))
	}

	return errors.Join(errs...)
}

type InitialPaymentSettlementStatus string

const (
	CreatedInitialPaymentSettlementStatus    InitialPaymentSettlementStatus = "created"
	AuthorizedInitialPaymentSettlementStatus InitialPaymentSettlementStatus = "authorized"
	SettledInitialPaymentSettlementStatus    InitialPaymentSettlementStatus = "settled"
)

func (s InitialPaymentSettlementStatus) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid payment settlement status: %s", s)
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
	GenericSettlement

	InitialStatus InitialPaymentSettlementStatus `json:"status"`
}

func (s ExternalSettlement) Validate() error {
	var errs []error

	if err := s.InitialStatus.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("initial status: %w", err))
	}

	if err := s.GenericSettlement.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type PromotionalSettlement struct{}

func (s PromotionalSettlement) Validate() error {
	return nil
}

type Settlement struct {
	t SettlementType

	invoice     *InvoiceSettlement
	external    *ExternalSettlement
	promotional *PromotionalSettlement
}

func (s Settlement) MarshalJSON() ([]byte, error) {
	var serde interface{}

	switch s.t {
	case SettlementTypeInvoice:
		serde = struct {
			Type SettlementType `json:"type"`
			*InvoiceSettlement
		}{
			Type:              SettlementTypeInvoice,
			InvoiceSettlement: s.invoice,
		}
	case SettlementTypeExternal:
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
		v := &InvoiceSettlement{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize InvoiceCreditPurchaseSettlement: %w", err)
		}

		s.invoice = v
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

func NewSettlement[T InvoiceSettlement | ExternalSettlement | PromotionalSettlement](settlement T) Settlement {
	switch v := any(settlement).(type) {
	case InvoiceSettlement:
		return Settlement{
			t:       SettlementTypeInvoice,
			invoice: &v,
		}
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
		if s.invoice == nil {
			return fmt.Errorf("invoice is required")
		}

		if err := s.invoice.Validate(); err != nil {
			return fmt.Errorf("invoice: %w", err)
		}
	case SettlementTypeExternal:
		if s.external == nil {
			return fmt.Errorf("external is required")
		}

		if err := s.external.Validate(); err != nil {
			return fmt.Errorf("external: %w", err)
		}
	case SettlementTypePromotional:
		if s.promotional == nil {
			return fmt.Errorf("promotional is required")
		}

		if err := s.promotional.Validate(); err != nil {
			return fmt.Errorf("promotional: %w", err)
		}
	default:
		return fmt.Errorf("invalid credit purchase settlement type: %s", s.t)
	}
	return nil
}

func (s Settlement) AsInvoiceSettlement() (InvoiceSettlement, error) {
	if s.t != SettlementTypeInvoice {
		return InvoiceSettlement{}, fmt.Errorf("settlement is not an invoice settlement")
	}

	if s.invoice == nil {
		return InvoiceSettlement{}, fmt.Errorf("invoice is nil")
	}

	return *s.invoice, nil
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

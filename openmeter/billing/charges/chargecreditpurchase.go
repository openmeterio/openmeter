package charges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type CreditPurchaseCharge struct {
	ManagedResource

	Status ChargeStatus `json:"status"`

	Intent CreditPurchaseIntent `json:"intent"`
	State  CreditPurchaseState  `json:"state"`
}

func (c CreditPurchaseCharge) Validate() error {
	var errs []error

	if err := c.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := c.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	if err := c.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	return errors.Join(errs...)
}

func (c CreditPurchaseCharge) AsCharge() Charge {
	return Charge{
		t:              ChargeTypeCreditPurchase,
		creditPurchase: &c,
	}
}

type CreditPurchaseIntent struct {
	IntentMeta

	CreditAmount alpacadecimal.Decimal `json:"amount"`

	// Settlement intent
	Settlement CreditPurchaseSettlement `json:"settlement"`
}

func (i CreditPurchaseIntent) Validate() error {
	var errs []error

	if err := i.IntentMeta.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent meta: %w", err))
	}

	if !i.CreditAmount.IsPositive() {
		errs = append(errs, fmt.Errorf("credit amount must be positive"))
	}

	if err := i.Settlement.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement: %w", err))
	}

	return errors.Join(errs...)
}

type CreditPurchaseSettlementType string

const (
	CreditPurchaseSettlementTypeInvoice     CreditPurchaseSettlementType = "invoice"
	CreditPurchaseSettlementTypeExternal    CreditPurchaseSettlementType = "external"
	CreditPurchaseSettlementTypePromotional CreditPurchaseSettlementType = "promotional"
)

func (s CreditPurchaseSettlementType) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid credit purchase settlement type: %s", s)
	}
	return nil
}

func (s CreditPurchaseSettlementType) Values() []string {
	return []string{
		string(CreditPurchaseSettlementTypeInvoice),
		string(CreditPurchaseSettlementTypeExternal),
		string(CreditPurchaseSettlementTypePromotional),
	}
}

type GenericCreditPurchaseSettlement struct {
	SettlementCurrency currencyx.Code            `json:"settlementCurrency"`
	CostBasis          alpacadecimal.Decimal     `json:"costBasis"`
	TaxConfig          *productcatalog.TaxConfig `json:"taxConfig"`
}

func (s GenericCreditPurchaseSettlement) Validate() error {
	var errs []error

	if err := s.SettlementCurrency.Validate(); err != nil {
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

type InvoiceCreditPurchaseSettlement struct {
	GenericCreditPurchaseSettlement
}

func (s InvoiceCreditPurchaseSettlement) Validate() error {
	var errs []error

	if err := s.GenericCreditPurchaseSettlement.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("generic credit purchase settlement: %w", err))
	}

	return errors.Join(errs...)
}

type PaymentSettlementStatus string

const (
	InitiatedPaymentSettlementStatus  PaymentSettlementStatus = "initiated"
	AuthorizedPaymentSettlementStatus PaymentSettlementStatus = "authorized"
	SettledPaymentSettlementStatus    PaymentSettlementStatus = "settled"
)

func (s PaymentSettlementStatus) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid payment settlement status: %s", s)
	}
	return nil
}

func (s PaymentSettlementStatus) Values() []string {
	return []string{
		string(InitiatedPaymentSettlementStatus),
		string(AuthorizedPaymentSettlementStatus),
		string(SettledPaymentSettlementStatus),
	}
}

type ExternalAuthorizedCreditPurchaseSettlement struct {
	GenericCreditPurchaseSettlement

	InitialStatus PaymentSettlementStatus `json:"status"`
}

func (s ExternalAuthorizedCreditPurchaseSettlement) Validate() error {
	var errs []error

	if err := s.InitialStatus.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("initial status: %w", err))
	}

	if err := s.GenericCreditPurchaseSettlement.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("generic credit purchase settlement: %w", err))
	}

	return errors.Join(errs...)
}

type PromotionalCreditPurchaseSettlement struct{}

func (s PromotionalCreditPurchaseSettlement) Validate() error {
	return nil
}

type CreditPurchaseSettlement struct {
	t CreditPurchaseSettlementType

	invoice     *InvoiceCreditPurchaseSettlement
	external    *ExternalAuthorizedCreditPurchaseSettlement
	promotional *PromotionalCreditPurchaseSettlement
}

func (s CreditPurchaseSettlement) MarshalJSON() ([]byte, error) {
	var serde interface{}

	switch s.t {
	case CreditPurchaseSettlementTypeInvoice:
		serde = struct {
			Type CreditPurchaseSettlementType `json:"type"`
			*InvoiceCreditPurchaseSettlement
		}{
			Type:                            CreditPurchaseSettlementTypeInvoice,
			InvoiceCreditPurchaseSettlement: s.invoice,
		}
	case CreditPurchaseSettlementTypeExternal:
		serde = struct {
			Type CreditPurchaseSettlementType `json:"type"`
			*ExternalAuthorizedCreditPurchaseSettlement
		}{
			Type: CreditPurchaseSettlementTypeExternal,
			ExternalAuthorizedCreditPurchaseSettlement: s.external,
		}
	case CreditPurchaseSettlementTypePromotional:
		serde = struct {
			Type CreditPurchaseSettlementType `json:"type"`
		}{
			Type: CreditPurchaseSettlementTypePromotional,
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

func (s *CreditPurchaseSettlement) UnmarshalJSON(bytes []byte) error {
	serde := &struct {
		Type CreditPurchaseSettlementType `json:"type"`
	}{}

	if err := json.Unmarshal(bytes, serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize CreditPurchaseSettlement type: %w", err)
	}

	switch serde.Type {
	case CreditPurchaseSettlementTypeInvoice:
		v := &InvoiceCreditPurchaseSettlement{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize InvoiceCreditPurchaseSettlement: %w", err)
		}

		s.invoice = v
		s.t = CreditPurchaseSettlementTypeInvoice
	case CreditPurchaseSettlementTypeExternal:
		v := &ExternalAuthorizedCreditPurchaseSettlement{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize ExternalAuthorizedCreditPurchaseSettlement: %w", err)
		}

		s.external = v
		s.t = CreditPurchaseSettlementTypeExternal
	case CreditPurchaseSettlementTypePromotional:
		s.promotional = &PromotionalCreditPurchaseSettlement{}
		s.t = CreditPurchaseSettlementTypePromotional
	default:
		return fmt.Errorf("invalid credit purchase settlement type: %s", serde.Type)
	}

	return nil
}

func NewCreditPurchaseSettlement[T InvoiceCreditPurchaseSettlement | ExternalAuthorizedCreditPurchaseSettlement | PromotionalCreditPurchaseSettlement](settlement T) CreditPurchaseSettlement {
	switch v := any(settlement).(type) {
	case InvoiceCreditPurchaseSettlement:
		return CreditPurchaseSettlement{
			t:       CreditPurchaseSettlementTypeInvoice,
			invoice: &v,
		}
	case ExternalAuthorizedCreditPurchaseSettlement:
		return CreditPurchaseSettlement{
			t:        CreditPurchaseSettlementTypeExternal,
			external: &v,
		}
	case PromotionalCreditPurchaseSettlement:
		return CreditPurchaseSettlement{
			t:           CreditPurchaseSettlementTypePromotional,
			promotional: &v,
		}
	default:
		return CreditPurchaseSettlement{}
	}
}

func (s CreditPurchaseSettlement) Type() CreditPurchaseSettlementType {
	return s.t
}

func (s CreditPurchaseSettlement) Validate() error {
	switch s.t {
	case CreditPurchaseSettlementTypeInvoice:
		if s.invoice == nil {
			return fmt.Errorf("invoice is required")
		}

		if err := s.invoice.Validate(); err != nil {
			return fmt.Errorf("invoice: %w", err)
		}
	case CreditPurchaseSettlementTypeExternal:
		if s.external == nil {
			return fmt.Errorf("external is required")
		}

		if err := s.external.Validate(); err != nil {
			return fmt.Errorf("external: %w", err)
		}
	case CreditPurchaseSettlementTypePromotional:
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

func (s CreditPurchaseSettlement) AsInvoiceCreditPurchaseSettlement() (InvoiceCreditPurchaseSettlement, error) {
	if s.t != CreditPurchaseSettlementTypeInvoice {
		return InvoiceCreditPurchaseSettlement{}, fmt.Errorf("credit purchase settlement is not an invoice credit purchase settlement")
	}

	if s.invoice == nil {
		return InvoiceCreditPurchaseSettlement{}, fmt.Errorf("invoice is nil")
	}

	return *s.invoice, nil
}

func (s CreditPurchaseSettlement) AsExternalAuthorizedCreditPurchaseSettlement() (ExternalAuthorizedCreditPurchaseSettlement, error) {
	if s.t != CreditPurchaseSettlementTypeExternal {
		return ExternalAuthorizedCreditPurchaseSettlement{}, fmt.Errorf("credit purchase settlement is not an external authorized credit purchase settlement")
	}

	if s.external == nil {
		return ExternalAuthorizedCreditPurchaseSettlement{}, fmt.Errorf("external is nil")
	}

	return *s.external, nil
}

func (s CreditPurchaseSettlement) AsPromotionalCreditPurchaseSettlement() (PromotionalCreditPurchaseSettlement, error) {
	if s.t != CreditPurchaseSettlementTypePromotional {
		return PromotionalCreditPurchaseSettlement{}, fmt.Errorf("credit purchase settlement is not a promotional credit purchase settlement")
	}

	if s.promotional == nil {
		return PromotionalCreditPurchaseSettlement{}, fmt.Errorf("promotional is nil")
	}

	return *s.promotional, nil
}

type CreditPurchaseState struct {
	Status PaymentSettlementStatus `json:"status"`
}

func (s CreditPurchaseState) Validate() error {
	var errs []error

	if err := s.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	return errors.Join(errs...)
}

// CreditPurchaseHandler is the interface for handling credit purchase charges.
// It is used to handle the different types of credit purchase charges (promotional, external, invoice).
//
// Promotional credit purchases are handled by the OnPromotionalCreditPurchase method only.
//
// Cost basis > 0 credit purchases are handled by the OnCreditPurchase method, which is the initial call.
// Happy path:
// - OnCreditPurchase is called
// - OnCreditPurchasePaymentAuthorized is called
// - OnCreditPurchasePaymentSettled is called
//
// Failed payment can occur either after the OnCreditPurchase or after the OnCreditPurchasePaymentAuthorized call.

type CreditPurchaseHandler interface {
	// Promotional credit handler methods (cost basis == 0)
	// ----------------------------------------------------

	// OnPromotionalCreditPurchase is called when a promotional credit purchase is created (e.g. costbasis is 0)
	// For promotional credit purchases we don't call any of the payment handler methods.
	OnPromotionalCreditPurchase(ctx context.Context, charge CreditPurchaseCharge) ([]CreditRealizationCreateInput, error)

	// Credit purchase handler methods (cost basis > 0)
	// ------------------------------------------------

	// OnCreditPurchase is called when a credit purchase is initiated that is going to be settled by
	// a payment (either external or a standard invoice)
	// Initial call
	OnCreditPurchase(ctx context.Context, charge CreditPurchaseCharge) ([]CreditRealizationCreateInput, error)

	// OnCreditPurchasePaymentAuthorized is called when a credit purchase payment is authorized for a credit
	// purchase.
	OnCreditPurchasePaymentAuthorized(ctx context.Context, charge CreditPurchaseCharge) (LedgerTransactionGroupReference, error)

	// OnCreditPurchasePaymentSettled is called when a credit purchase payment is settled for a credit
	// purchase.
	OnCreditPurchasePaymentSettled(ctx context.Context, charge CreditPurchaseCharge) (LedgerTransactionGroupReference, error)

	// OnCreditPurchasePaymentUncollectible is called when a credit purchase payment is uncollectible
	OnCreditPurchasePaymentUncollectible(ctx context.Context, charge CreditPurchaseCharge) (LedgerTransactionGroupReference, error)
}

package plan

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
)

const DefaultPaymentTerm = InAdvancePaymentTerm

const (
	InAdvancePaymentTerm PaymentTermType = "in_advance"
	InArrearsPaymentTerm PaymentTermType = "in_arrears"
)

type PaymentTermType string

func (p PaymentTermType) Values() []PaymentTermType {
	return []PaymentTermType{
		InAdvancePaymentTerm,
		InArrearsPaymentTerm,
	}
}

func (p PaymentTermType) StringValues() []string {
	return []string{
		string(InAdvancePaymentTerm),
		string(InArrearsPaymentTerm),
	}
}

const (
	FlatPriceType   PriceType = "flat"
	UnitPriceType   PriceType = "unit"
	TieredPriceType PriceType = "tiered"
)

type PriceType string

func (p PriceType) Values() []string {
	return []string{
		string(FlatPriceType),
		string(UnitPriceType),
		string(TieredPriceType),
	}
}

type pricer interface {
	json.Marshaler
	json.Unmarshaler
	Validator

	Type() PriceType
	AsFlat() (FlatPrice, error)
	AsUnit() (UnitPrice, error)
	AsTiered() (TieredPrice, error)
	FromFlat(FlatPrice)
	FromUnit(UnitPrice)
	FromTiered(TieredPrice)
}

var _ pricer = (*Price)(nil)

type Price struct {
	t      PriceType
	flat   *FlatPrice
	unit   *UnitPrice
	tiered *TieredPrice
}

func (p *Price) MarshalJSON() ([]byte, error) {
	var b []byte
	var err error

	switch p.t {
	case FlatPriceType:
		b, err = json.Marshal(p.flat)
		if err != nil {
			return nil, fmt.Errorf("failed to json marshal FlatPrice: %w", err)
		}
	case UnitPriceType:
		b, err = json.Marshal(p.unit)
		if err != nil {
			return nil, fmt.Errorf("failed to json marshal UnitPrice: %w", err)
		}
	case TieredPriceType:
		b, err = json.Marshal(p.tiered)
		if err != nil {
			return nil, fmt.Errorf("failed to json marshal TieredPrice: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid type: %s", p.t)
	}

	return b, nil
}

func (p *Price) UnmarshalJSON(bytes []byte) error {
	meta := &PriceMeta{}

	if err := json.Unmarshal(bytes, meta); err != nil {
		return fmt.Errorf("failed to json unmarshal Price type: %w", err)
	}

	switch meta.Type {
	case FlatPriceType:
		v := &FlatPrice{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal FlatPrice: %w", err)
		}

		p.flat = v
		p.t = FlatPriceType
	case UnitPriceType:
		v := &UnitPrice{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal UnitPrice: %w", err)
		}

		p.unit = v
		p.t = UnitPriceType
	case TieredPriceType:
		v := &TieredPrice{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal TieredPrice: %w", err)
		}

		p.tiered = v
		p.t = TieredPriceType
	default:
		return fmt.Errorf("invalid type: %s", meta.Type)
	}

	return nil
}

func (p *Price) Validate() error {
	switch p.t {
	case FlatPriceType:
		return p.flat.Validate()
	case UnitPriceType:
		return p.unit.Validate()
	case TieredPriceType:
		return p.tiered.Validate()
	default:
		return errors.New("invalid Price: not initialized")
	}
}

func (p *Price) Type() PriceType {
	return p.t
}

func (p *Price) AsFlat() (FlatPrice, error) {
	if p.t == "" || p.flat == nil {
		return FlatPrice{}, errors.New("invalid FlatPrice: not initialized")
	}

	if p.t != FlatPriceType {
		return FlatPrice{}, fmt.Errorf("type mismatch: %s", p.t)
	}

	return *p.flat, nil
}

func (p *Price) AsUnit() (UnitPrice, error) {
	if p.t == "" || p.unit == nil {
		return UnitPrice{}, errors.New("invalid UnitPrice: not initialized")
	}

	if p.t != UnitPriceType {
		return UnitPrice{}, fmt.Errorf("type mismatch: %s", p.t)
	}

	return *p.unit, nil
}

func (p *Price) AsTiered() (TieredPrice, error) {
	if p.t == "" || p.tiered == nil {
		return TieredPrice{}, errors.New("invalid TieredPrice: not initialized")
	}

	if p.t != TieredPriceType {
		return TieredPrice{}, fmt.Errorf("type mismatch: %s", p.t)
	}

	return *p.tiered, nil
}

func (p *Price) FromFlat(price FlatPrice) {
	p.flat = &price
	p.t = FlatPriceType
	p.flat.Type = UnitPriceType
}

func (p *Price) FromUnit(price UnitPrice) {
	p.unit = &price
	p.t = UnitPriceType
	p.unit.Type = UnitPriceType
}

func (p *Price) FromTiered(price TieredPrice) {
	p.tiered = &price
	p.t = TieredPriceType
	p.tiered.Type = TieredPriceType
}

func NewPriceFrom[T FlatPrice | UnitPrice | TieredPrice](v T) Price {
	p := Price{}

	switch any(v).(type) {
	case FlatPrice:
		flat := any(v).(FlatPrice)
		p.FromFlat(flat)
	case UnitPrice:
		unit := any(v).(UnitPrice)
		p.FromUnit(unit)
	case TieredPrice:
		tiered := any(v).(TieredPrice)
		p.FromTiered(tiered)
	}

	return p
}

type PriceMeta struct {
	// Type of the price.
	Type PriceType `json:"type"`
}

type FlatPrice struct {
	PriceMeta

	// Amount of the flat price.
	Amount decimal.Decimal `json:"amount"`

	// PaymentTerm defines the payment term of the flat price.
	// Defaults to InAdvancePaymentTerm.
	PaymentTerm PaymentTermType `json:"payment_term,omitempty"`
}

func (f FlatPrice) Validate() error {
	var errs []error

	if f.Amount.IsNegative() {
		errs = append(errs, errors.New("the Amount must not be negative"))
	}

	if !lo.Contains(PaymentTermType("").Values(), f.PaymentTerm) {
		errs = append(errs, fmt.Errorf("invalid PaymentTerm: %s", f.PaymentTerm))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type UnitPrice struct {
	PriceMeta

	// Amount of the unit price.
	Amount decimal.Decimal `json:"amount"`

	// MinimumAmount defines the least amount the customer committed to spend.
	MinimumAmount *decimal.Decimal `json:"minimumAmount,omitempty"`

	// MaximumAmount defines the upper limit of amount the customer entitled to spend.
	MaximumAmount *decimal.Decimal `json:"maximumAmount,omitempty"`
}

func (u UnitPrice) Validate() error {
	var errs []error

	if u.Amount.IsNegative() {
		errs = append(errs, errors.New("the Amount must not be negative"))
	}

	minAmount := lo.FromPtrOr(u.MinimumAmount, decimal.Zero)
	if minAmount.IsNegative() {
		errs = append(errs, errors.New("the MinimumAmount must not be negative"))
	}

	maxAmount := lo.FromPtrOr(u.MaximumAmount, decimal.Zero)
	if maxAmount.IsNegative() {
		errs = append(errs, errors.New("the MaximumAmount must not be negative"))
	}

	if !minAmount.IsZero() && !maxAmount.IsZero() {
		if minAmount.GreaterThan(maxAmount) {
			errs = append(errs, errors.New("the MinimumAmount must not be greater than MaximumAmount"))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

const (
	VolumeTieredPrice    TieredPriceMode = "volume"
	GraduatedTieredPrice TieredPriceMode = "graduated"
)

type TieredPriceMode string

func (p TieredPriceMode) Values() []TieredPriceMode {
	return []TieredPriceMode{
		VolumeTieredPrice,
		GraduatedTieredPrice,
	}
}

func (p TieredPriceMode) StringValues() []string {
	return []string{
		string(VolumeTieredPrice),
		string(GraduatedTieredPrice),
	}
}

func NewTieredPriceMode(s string) (TieredPriceMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(VolumeTieredPrice):
		return VolumeTieredPrice, nil
	case string(GraduatedTieredPrice):
		return GraduatedTieredPrice, nil
	default:
		return "", fmt.Errorf("invalid TieredPrice mode: %s", s)
	}
}

type TieredPrice struct {
	PriceMeta

	// Mode defines whether the tier is volume-based or graduated.
	// * VolumeTieredPrice: the maximum quantity within a period determines the per-unit price
	// * GraduatedTieredPrice: pricing can change as the quantity grows
	Mode TieredPriceMode `json:"mode"`

	// Tiers defines the list of PriceTier.
	Tiers []PriceTier `json:"tiers"`

	// MinimumAmount defines the least amount the customer committed to spend.
	MinimumAmount *decimal.Decimal `json:"minimumAmount,omitempty"`

	// MaximumAmount defines the upper limit of amount the customer entitled to spend.
	MaximumAmount *decimal.Decimal `json:"maximumAmount,omitempty"`
}

func (t TieredPrice) Validate() error {
	var errs []error

	if !lo.Contains(TieredPriceMode("").Values(), t.Mode) {
		errs = append(errs, fmt.Errorf("invalid TieredPrice mode: %s", t.Mode))
	}

	upToAmounts := make(map[string]struct{}, len(t.Tiers))
	for _, tier := range t.Tiers {
		uta := lo.FromPtrOr(tier.UpToAmount, decimal.Zero)
		if !uta.IsZero() {
			if _, ok := upToAmounts[uta.String()]; ok {
				errs = append(errs, errors.New("multiple PriceTiers with same UpToAmount are not allowed"))

				continue
			}
			upToAmounts[uta.String()] = struct{}{}
		}

		if err := tier.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid PriceTier: %w", err))
		}
	}

	minAmount := lo.FromPtrOr(t.MinimumAmount, decimal.Zero)
	if minAmount.IsNegative() {
		errs = append(errs, errors.New("the MinimumAmount must not be negative"))
	}

	maxAmount := lo.FromPtrOr(t.MaximumAmount, decimal.Zero)
	if maxAmount.IsNegative() {
		errs = append(errs, errors.New("the MaximumAmount must not be negative"))
	}

	if !minAmount.IsZero() && !maxAmount.IsZero() {
		if minAmount.GreaterThan(maxAmount) {
			errs = append(errs, errors.New("minimum amount must not be greater than maximum amount"))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

var _ Validator = (*PriceTier)(nil)

// PriceTier describes a tier of price(s).
type PriceTier struct {
	// UpToAmount defines the quantity will be contained in the tier. Inclusive.
	// If null, the tier is open-ended.
	UpToAmount *decimal.Decimal `json:"upToAmount,omitempty"`

	// FlatPrice defines the flat price component of the tier.
	FlatPrice *PriceTierFlatPrice `json:"flatPrice,omitempty"`

	// UnitPrice defines the unit price component of the tier.
	UnitPrice *PriceTierUnitPrice `json:"unitPrice,omitempty"`
}

func (p PriceTier) Validate() error {
	var errs []error

	upToAmount := lo.FromPtrOr(p.UpToAmount, decimal.Zero)
	if upToAmount.IsNegative() {
		errs = append(errs, errors.New("the UpToAmount must not be negative"))
	}

	if p.FlatPrice == nil && p.UnitPrice == nil {
		errs = append(errs, errors.New("either FlatPrice or UnitPrice must be provided in PriceTier"))
	}

	if p.FlatPrice != nil {
		if err := p.FlatPrice.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid FlatPrice in PriceTier: %w", err))
		}
	}

	if p.UnitPrice != nil {
		if err := p.UnitPrice.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid UnitPrice in PriceTier: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

var _ Validator = (*PriceTierFlatPrice)(nil)

type PriceTierFlatPrice struct {
	// Amount of the flat price.
	Amount decimal.Decimal `json:"amount"`
}

func (f PriceTierFlatPrice) Validate() error {
	if f.Amount.IsNegative() {
		return errors.New("the Amount must not be negative")
	}

	return nil
}

var _ Validator = (*PriceTierUnitPrice)(nil)

type PriceTierUnitPrice struct {
	// Amount of the flat price.
	Amount decimal.Decimal `json:"amount"`
}

func (u PriceTierUnitPrice) Validate() error {
	if u.Amount.IsNegative() {
		return errors.New("the Amount must not be negative")
	}

	return nil
}

package productcatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
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
	models.Validator

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
	var serde interface{}

	switch p.t {
	case FlatPriceType:
		serde = &struct {
			Type PriceType `json:"type"`
			*FlatPrice
		}{
			Type:      p.t,
			FlatPrice: p.flat,
		}
	case UnitPriceType:
		serde = &struct {
			Type PriceType `json:"type"`
			*UnitPrice
		}{
			Type:      p.t,
			UnitPrice: p.unit,
		}
	case TieredPriceType:
		serde = &struct {
			Type PriceType `json:"type"`
			*TieredPrice
		}{
			Type:        p.t,
			TieredPrice: p.tiered,
		}
	default:
		return nil, fmt.Errorf("invalid Price type: %s", p.t)
	}

	b, err = json.Marshal(serde)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON serialize Price: %w", err)
	}

	return b, nil
}

func (p *Price) UnmarshalJSON(bytes []byte) error {
	serde := &struct {
		Type PriceType `json:"type"`
	}{}

	if err := json.Unmarshal(bytes, serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize Price type: %w", err)
	}

	switch serde.Type {
	case FlatPriceType:
		v := &FlatPrice{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize FlatPrice: %w", err)
		}

		p.flat = v
		p.t = FlatPriceType
	case UnitPriceType:
		v := &UnitPrice{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize UnitPrice: %w", err)
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
		return fmt.Errorf("invalid Price type: %s", serde.Type)
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

func (p *Price) Equal(v *Price) bool {
	if p == nil && v == nil {
		return true
	}

	if p == nil || v == nil {
		return false
	}

	if p.t != v.t {
		return false
	}
	switch p.t {
	case FlatPriceType:
		return p.flat.Equal(v.flat)
	case UnitPriceType:
		return p.unit.Equal(v.unit)
	case TieredPriceType:
		return p.tiered.Equal(v.tiered)
	default:
		return false
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
}

func (p *Price) FromUnit(price UnitPrice) {
	p.unit = &price
	p.t = UnitPriceType
}

func (p *Price) FromTiered(price TieredPrice) {
	p.tiered = &price
	p.t = TieredPriceType
}

func NewPriceFrom[T FlatPrice | UnitPrice | TieredPrice](v T) *Price {
	p := &Price{}

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

type FlatPrice struct {
	// Amount of the flat price.
	Amount decimal.Decimal `json:"amount"`

	// PaymentTerm defines the payment term of the flat price.
	// Defaults to InAdvancePaymentTerm.
	PaymentTerm PaymentTermType `json:"paymentTerm,omitempty"`
}

func (f *FlatPrice) Equal(v *FlatPrice) bool {
	if f == nil && v == nil {
		return true
	}

	if f == nil || v == nil {
		return false
	}

	if !f.Amount.Equal(v.Amount) {
		return false
	}

	if f.PaymentTerm != v.PaymentTerm {
		return false
	}

	return true
}

func (f *FlatPrice) Validate() error {
	var errs []error

	if f.Amount.IsNegative() {
		errs = append(errs, errors.New("the Amount must not be negative"))
	}

	if !lo.Contains(PaymentTermType("").Values(), f.PaymentTerm) {
		errs = append(errs, fmt.Errorf("invalid PaymentTerm: %s", f.PaymentTerm))
	}

	return NewValidationError(errors.Join(errs...))
}

type UnitPrice struct {
	// Amount of the unit price.
	Amount decimal.Decimal `json:"amount"`

	// MinimumAmount defines the least amount the customer committed to spend.
	MinimumAmount *decimal.Decimal `json:"minimumAmount,omitempty"`

	// MaximumAmount defines the upper limit of amount the customer entitled to spend.
	MaximumAmount *decimal.Decimal `json:"maximumAmount,omitempty"`
}

func (u *UnitPrice) Equal(v *UnitPrice) bool {
	if u == nil && v == nil {
		return true
	}

	if u == nil || v == nil {
		return false
	}

	if !u.Amount.Equal(v.Amount) {
		return false
	}

	if u.MinimumAmount != nil && v.MinimumAmount == nil {
		return false
	}

	if u.MinimumAmount == nil && v.MinimumAmount != nil {
		return false
	}

	if !lo.FromPtr(u.MinimumAmount).Equal(lo.FromPtr(v.MinimumAmount)) {
		return false
	}

	if u.MaximumAmount != nil && v.MaximumAmount == nil {
		return false
	}

	if u.MaximumAmount == nil && v.MaximumAmount != nil {
		return false
	}

	if !lo.FromPtr(u.MaximumAmount).Equal(lo.FromPtr(v.MaximumAmount)) {
		return false
	}

	return true
}

func (u *UnitPrice) Validate() error {
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

	return NewValidationError(errors.Join(errs...))
}

const (
	// In `volume`-based tiering, the maximum quantity within a period determines the per unit price.
	VolumeTieredPrice TieredPriceMode = "volume"
	// In `graduated` tiering, pricing can change as the quantity grows.
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

func (t *TieredPrice) Equal(v *TieredPrice) bool {
	if t == nil && v == nil {
		return true
	}

	if t == nil || v == nil {
		return false
	}

	if t.Mode != v.Mode {
		return false
	}

	if len(t.Tiers) != len(v.Tiers) {
		return false
	}

	if t.MinimumAmount == nil && v.MinimumAmount != nil {
		return false
	}

	if !lo.FromPtr(t.MinimumAmount).Equal(lo.FromPtr(v.MinimumAmount)) {
		return false
	}

	if t.MaximumAmount != nil && v.MaximumAmount == nil {
		return false
	}

	if t.MaximumAmount == nil && v.MaximumAmount != nil {
		return false
	}

	if !lo.FromPtr(t.MaximumAmount).Equal(lo.FromPtr(v.MaximumAmount)) {
		return false
	}

	return true
}

func (t *TieredPrice) Validate() error {
	var errs []error

	if !lo.Contains(TieredPriceMode("").Values(), t.Mode) {
		errs = append(errs, fmt.Errorf("invalid TieredPrice mode: %s", t.Mode))
	}

	if len(t.Tiers) == 0 {
		errs = append(errs, errors.New("at least one PriceTier must be provided"))
	}

	upToAmounts := make(map[string]struct{}, len(t.Tiers))
	tierOpenEndedPresent := false
	for _, tier := range t.Tiers {
		if tier.UpToAmount == nil {
			tierOpenEndedPresent = true
		}

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

	if !tierOpenEndedPresent {
		errs = append(errs, errors.New("at least one PriceTier must be open-ended"))
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

	return NewValidationError(errors.Join(errs...))
}

func (t *TieredPrice) WithSortedTiers() TieredPrice {
	out := *t
	out.Tiers = make([]PriceTier, len(t.Tiers))
	copy(out.Tiers, t.Tiers)

	// Sort tiers by UpToAmount in ascending order
	slices.SortFunc(out.Tiers, func(a, b PriceTier) int {
		if a.UpToAmount == nil && b.UpToAmount == nil {
			return 0
		}

		if a.UpToAmount == nil {
			return 1
		}

		if b.UpToAmount == nil {
			return -1
		}

		return a.UpToAmount.Cmp(*b.UpToAmount)
	})

	return out
}

var _ models.Validator = (*PriceTier)(nil)

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

	return NewValidationError(errors.Join(errs...))
}

var _ models.Validator = (*PriceTierFlatPrice)(nil)

type PriceTierFlatPrice struct {
	// Amount of the flat price.
	Amount decimal.Decimal `json:"amount"`
}

func (f PriceTierFlatPrice) Validate() error {
	if f.Amount.IsNegative() {
		return NewValidationError(errors.New("the Amount must not be negative"))
	}

	return nil
}

var _ models.Validator = (*PriceTierUnitPrice)(nil)

type PriceTierUnitPrice struct {
	// Amount of the flat price.
	Amount decimal.Decimal `json:"amount"`
}

func (u PriceTierUnitPrice) Validate() error {
	if u.Amount.IsNegative() {
		return NewValidationError(errors.New("the Amount must not be negative"))
	}

	return nil
}

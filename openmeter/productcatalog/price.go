package productcatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/models"
)

const DefaultPaymentTerm = InAdvancePaymentTerm

const (
	InAdvancePaymentTerm PaymentTermType = "in_advance"
	InArrearsPaymentTerm PaymentTermType = "in_arrears"
)

type PaymentTermType string

func (p PaymentTermType) Values() []string {
	return []string{
		string(InAdvancePaymentTerm),
		string(InArrearsPaymentTerm),
	}
}

func (p PaymentTermType) StringValues() []string {
	return []string{
		string(InAdvancePaymentTerm),
		string(InArrearsPaymentTerm),
	}
}

func (p PaymentTermType) Validate() error {
	if !slices.Contains(p.Values(), string(p)) {
		return fmt.Errorf("invalid payment term type: %s", p)
	}

	return nil
}

const (
	FlatPriceType    PriceType = "flat"
	UnitPriceType    PriceType = "unit"
	TieredPriceType  PriceType = "tiered"
	DynamicPriceType PriceType = "dynamic"
	PackagePriceType PriceType = "package"
)

type PriceType string

func (p PriceType) Values() []string {
	return []string{
		string(FlatPriceType),
		string(UnitPriceType),
		string(TieredPriceType),
		string(DynamicPriceType),
		string(PackagePriceType),
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
	AsDynamic() (DynamicPrice, error)
	AsPackage() (PackagePrice, error)
	FromFlat(FlatPrice)
	FromUnit(UnitPrice)
	FromTiered(TieredPrice)
	FromDynamic(DynamicPrice)
	FromPackage(PackagePrice)

	// Common field accessors
	// GetCommitments returns the commitments for the price, or an empty Commitments if the price type does not support commitments.
	GetCommitments() Commitments

	GetPaymentTerm() PaymentTermType
}

var _ pricer = (*Price)(nil)

type Price struct {
	t            PriceType
	flat         *FlatPrice
	unit         *UnitPrice
	tiered       *TieredPrice
	dynamic      *DynamicPrice
	packagePrice *PackagePrice
}

func (p *Price) Clone() *Price {
	clone := &Price{
		t: p.t,
	}

	switch p.t {
	case FlatPriceType:
		clone.flat = lo.ToPtr(p.flat.Clone())
	case UnitPriceType:
		clone.unit = lo.ToPtr(p.unit.Clone())
	case TieredPriceType:
		clone.tiered = lo.ToPtr(p.tiered.Clone())
	case DynamicPriceType:
		clone.dynamic = lo.ToPtr(p.dynamic.Clone())
	case PackagePriceType:
		clone.packagePrice = lo.ToPtr(p.packagePrice.Clone())
	}

	return clone
}

func (p Price) MarshalJSON() ([]byte, error) {
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
	case DynamicPriceType:
		serde = &struct {
			Type PriceType `json:"type"`
			*DynamicPrice
		}{
			Type:         p.t,
			DynamicPrice: p.dynamic,
		}
	case PackagePriceType:
		serde = &struct {
			Type PriceType `json:"type"`
			*PackagePrice
		}{
			Type:         p.t,
			PackagePrice: p.packagePrice,
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
	case DynamicPriceType:
		v := &DynamicPrice{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal DynamicPrice: %w", err)
		}

		p.dynamic = v
		p.t = DynamicPriceType
	case PackagePriceType:
		v := &PackagePrice{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal PackagePrice: %w", err)
		}

		p.packagePrice = v
		p.t = PackagePriceType
	default:
		return fmt.Errorf("invalid Price type: %s", serde.Type)
	}

	return nil
}

func (p *Price) Validate() error {
	if p == nil {
		return errors.New("invalid price: not initialized")
	}

	switch p.t {
	case FlatPriceType:
		return p.flat.Validate()
	case UnitPriceType:
		return p.unit.Validate()
	case TieredPriceType:
		return p.tiered.Validate()
	case DynamicPriceType:
		return p.dynamic.Validate()
	case PackagePriceType:
		return p.packagePrice.Validate()
	default:
		return fmt.Errorf("invalid price type: %q", p.t)
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
	case DynamicPriceType:
		return p.dynamic.Equal(v.dynamic)
	case PackagePriceType:
		return p.packagePrice.Equal(v.packagePrice)
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

func (p *Price) AsDynamic() (DynamicPrice, error) {
	if p.t == "" || p.dynamic == nil {
		return DynamicPrice{}, errors.New("invalid DynamicPrice: not initialized")
	}

	if p.t != DynamicPriceType {
		return DynamicPrice{}, fmt.Errorf("type mismatch: %s", p.t)
	}

	return *p.dynamic, nil
}

func (p *Price) AsPackage() (PackagePrice, error) {
	if p.t == "" || p.packagePrice == nil {
		return PackagePrice{}, errors.New("invalid PackagePrice: not initialized")
	}

	if p.t != PackagePriceType {
		return PackagePrice{}, fmt.Errorf("type mismatch: %s", p.t)
	}

	return *p.packagePrice, nil
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

func (p *Price) FromDynamic(price DynamicPrice) {
	p.dynamic = &price
	p.t = DynamicPriceType
}

func (p *Price) FromPackage(price PackagePrice) {
	p.packagePrice = &price
	p.t = PackagePriceType
}

func NewPriceFrom[T FlatPrice | UnitPrice | TieredPrice | DynamicPrice | PackagePrice](v T) *Price {
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
	case DynamicPrice:
		dynamic := any(v).(DynamicPrice)
		p.FromDynamic(dynamic)
	case PackagePrice:
		packagePrice := any(v).(PackagePrice)
		p.FromPackage(packagePrice)
	}

	return p
}

func (p *Price) GetCommitments() Commitments {
	switch p.t {
	case FlatPriceType:
		return Commitments{}
	case UnitPriceType:
		return p.unit.Commitments
	case TieredPriceType:
		return p.tiered.Commitments
	case DynamicPriceType:
		return p.dynamic.Commitments
	case PackagePriceType:
		return p.packagePrice.Commitments
	default:
		return Commitments{}
	}
}

func (p *Price) GetPaymentTerm() PaymentTermType {
	switch p.t {
	case FlatPriceType:
		// It's only an option for flat prices
		return p.flat.PaymentTerm
	}

	return InArrearsPaymentTerm
}

type FlatPrice struct {
	// Amount of the flat price.
	Amount decimal.Decimal `json:"amount"`

	// PaymentTerm defines the payment term of the flat price.
	// Defaults to InAdvancePaymentTerm.
	PaymentTerm PaymentTermType `json:"paymentTerm,omitempty"`
}

func (f *FlatPrice) Clone() FlatPrice {
	return FlatPrice{
		Amount:      f.Amount.Copy(),
		PaymentTerm: f.PaymentTerm,
	}
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

	if f.PaymentTerm != "" && !lo.Contains(PaymentTermType("").Values(), string(f.PaymentTerm)) {
		errs = append(errs, fmt.Errorf("invalid PaymentTerm: %s", f.PaymentTerm))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type UnitPrice struct {
	Commitments `json:",inline"`

	// Amount of the unit price.
	Amount decimal.Decimal `json:"amount"`
}

func (u *UnitPrice) Clone() UnitPrice {
	clone := UnitPrice{}

	if u.Commitments.MinimumAmount != nil {
		cp := u.Commitments.MinimumAmount.Copy()
		clone.Commitments.MinimumAmount = &cp
	}

	if u.Commitments.MaximumAmount != nil {
		cp := u.Commitments.MaximumAmount.Copy()
		clone.Commitments.MaximumAmount = &cp
	}

	clone.Amount = u.Amount.Copy()

	return clone
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

	if !u.Commitments.Equal(v.Commitments) {
		return false
	}

	return true
}

func (u *UnitPrice) Validate() error {
	var errs []error

	if u.Amount.IsNegative() {
		errs = append(errs, errors.New("the Amount must not be negative"))
	}

	if err := u.Commitments.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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
	Commitments `json:",inline"`

	// Mode defines whether the tier is volume-based or graduated.
	// * VolumeTieredPrice: the maximum quantity within a period determines the per-unit price
	// * GraduatedTieredPrice: pricing can change as the quantity grows
	Mode TieredPriceMode `json:"mode"`

	// Tiers defines the list of PriceTier.
	Tiers []PriceTier `json:"tiers"`
}

func (t *TieredPrice) Clone() TieredPrice {
	clone := TieredPrice{
		Mode: t.Mode,
	}

	if t.Commitments.MinimumAmount != nil {
		cp := t.Commitments.MinimumAmount.Copy()
		clone.Commitments.MinimumAmount = &cp
	}

	if t.Commitments.MaximumAmount != nil {
		cp := t.Commitments.MaximumAmount.Copy()
		clone.Commitments.MaximumAmount = &cp
	}

	tiersClone := make([]PriceTier, len(t.Tiers))
	for i, tier := range t.Tiers {
		tiersClone[i] = tier.Clone()
	}

	clone.Tiers = tiersClone

	return clone
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

	if !t.Commitments.Equal(v.Commitments) {
		return false
	}

	for i, tier := range t.Tiers {
		if !tier.Equal(v.Tiers[i]) {
			return false
		}
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

		uta := lo.FromPtr(tier.UpToAmount)
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

	if err := t.Commitments.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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

func (p PriceTier) Clone() PriceTier {
	clone := PriceTier{}

	if p.UpToAmount != nil {
		cp := p.UpToAmount.Copy()
		clone.UpToAmount = &cp
	}

	if p.FlatPrice != nil {
		clone.FlatPrice = lo.ToPtr(p.FlatPrice.Clone())
	}

	if p.UnitPrice != nil {
		clone.UnitPrice = lo.ToPtr(p.UnitPrice.Clone())
	}

	return clone
}

func (p PriceTier) Validate() error {
	var errs []error

	upToAmount := lo.FromPtr(p.UpToAmount)
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (p PriceTier) Equal(v PriceTier) bool {
	if !equal.PtrEqual(p.UpToAmount, v.UpToAmount) {
		return false
	}

	if !equal.PtrEqual(p.FlatPrice, v.FlatPrice) {
		return false
	}

	if !equal.PtrEqual(p.UnitPrice, v.UnitPrice) {
		return false
	}

	return true
}

var _ models.Validator = (*PriceTierFlatPrice)(nil)

type PriceTierFlatPrice struct {
	// Amount of the flat price.
	Amount decimal.Decimal `json:"amount"`
}

func (f PriceTierFlatPrice) Clone() PriceTierFlatPrice {
	return PriceTierFlatPrice{
		Amount: f.Amount.Copy(),
	}
}

func (f PriceTierFlatPrice) Validate() error {
	if f.Amount.IsNegative() {
		return models.NewGenericValidationError(errors.New("the Amount must not be negative"))
	}

	return nil
}

func (f PriceTierFlatPrice) Equal(v PriceTierFlatPrice) bool {
	return f.Amount.Equal(v.Amount)
}

var _ models.Validator = (*PriceTierUnitPrice)(nil)

type PriceTierUnitPrice struct {
	// Amount of the flat price.
	Amount decimal.Decimal `json:"amount"`
}

func (u PriceTierUnitPrice) Clone() PriceTierUnitPrice {
	return PriceTierUnitPrice{
		Amount: u.Amount.Copy(),
	}
}

func (u PriceTierUnitPrice) Validate() error {
	if u.Amount.IsNegative() {
		return models.NewGenericValidationError(errors.New("the Amount must not be negative"))
	}

	return nil
}

func (u PriceTierUnitPrice) Equal(v PriceTierUnitPrice) bool {
	return u.Amount.Equal(v.Amount)
}

type Commitments struct {
	// MinimumAmount defines the least amount the customer committed to spend.
	MinimumAmount *decimal.Decimal `json:"minimumAmount,omitempty"`

	// MaximumAmount defines the upper limit of amount the customer entitled to spend.
	MaximumAmount *decimal.Decimal `json:"maximumAmount,omitempty"`
}

func (c Commitments) Validate() error {
	var errs []error

	minAmount := lo.FromPtr(c.MinimumAmount)
	if minAmount.IsNegative() {
		errs = append(errs, errors.New("the MinimumAmount must not be negative"))
	}

	maxAmount := lo.FromPtr(c.MaximumAmount)
	if maxAmount.IsNegative() {
		errs = append(errs, errors.New("the MaximumAmount must not be negative"))
	}

	if !minAmount.IsZero() && !maxAmount.IsZero() {
		if minAmount.GreaterThan(maxAmount) {
			errs = append(errs, errors.New("the MinimumAmount must not be greater than MaximumAmount"))
		}
	}

	return errors.Join(errs...)
}

func (c Commitments) Equal(v Commitments) bool {
	if !equal.PtrEqual(c.MinimumAmount, v.MinimumAmount) {
		return false
	}

	if !equal.PtrEqual(c.MaximumAmount, v.MaximumAmount) {
		return false
	}

	return true
}

var _ models.Validator = (*DynamicPrice)(nil)

var DynamicPriceDefaultMultiplier = decimal.NewFromFloat(1)

type DynamicPrice struct {
	Commitments `json:",inline"`

	// Multiplier defines the multiplier applied to the price.
	Multiplier decimal.Decimal `json:"multiplier"`
}

func (p DynamicPrice) Clone() DynamicPrice {
	clone := DynamicPrice{}

	if p.Commitments.MinimumAmount != nil {
		cp := p.Commitments.MinimumAmount.Copy()
		clone.Commitments.MinimumAmount = &cp
	}

	if p.Commitments.MaximumAmount != nil {
		cp := p.Commitments.MaximumAmount.Copy()
		clone.Commitments.MaximumAmount = &cp
	}

	clone.Multiplier = p.Multiplier.Copy()

	return clone
}

func (p DynamicPrice) Validate() error {
	var errs []error

	if p.Multiplier.LessThan(decimal.Zero) {
		errs = append(errs, errors.New("the markup rate must not be less than 0"))
	}

	if err := p.Commitments.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (p *DynamicPrice) Equal(v *DynamicPrice) bool {
	if p == nil && v == nil {
		return true
	}

	if p == nil || v == nil {
		return false
	}

	if !p.Multiplier.Equal(v.Multiplier) {
		return false
	}

	if !p.Commitments.Equal(v.Commitments) {
		return false
	}

	return true
}

var _ models.Validator = (*PackagePrice)(nil)

type PackagePrice struct {
	Commitments `json:",inline"`

	Amount             decimal.Decimal `json:"amount"`
	QuantityPerPackage decimal.Decimal `json:"quantityPerPackage"`
}

func (p PackagePrice) Clone() PackagePrice {
	clone := PackagePrice{}

	if p.Commitments.MinimumAmount != nil {
		cp := p.Commitments.MinimumAmount.Copy()
		clone.Commitments.MinimumAmount = &cp
	}

	if p.Commitments.MaximumAmount != nil {
		cp := p.Commitments.MaximumAmount.Copy()
		clone.Commitments.MaximumAmount = &cp
	}

	clone.Amount = p.Amount.Copy()
	clone.QuantityPerPackage = p.QuantityPerPackage.Copy()

	return clone
}

func (p PackagePrice) Validate() error {
	var errs []error

	if p.Amount.IsNegative() {
		errs = append(errs, errors.New("the Amount must not be negative"))
	}

	if p.QuantityPerPackage.IsNegative() {
		errs = append(errs, errors.New("the QuantityPerPackage must not be negative"))
	}

	if p.QuantityPerPackage.IsZero() {
		errs = append(errs, errors.New("the QuantityPerPackage must not be zero"))
	}

	if err := p.Commitments.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (p *PackagePrice) Equal(v *PackagePrice) bool {
	if p == nil && v == nil {
		return true
	}

	if p == nil || v == nil {
		return false
	}

	if !p.Amount.Equal(v.Amount) {
		return false
	}

	if !p.QuantityPerPackage.Equal(v.QuantityPerPackage) {
		return false
	}

	if !p.Commitments.Equal(v.Commitments) {
		return false
	}

	return true
}

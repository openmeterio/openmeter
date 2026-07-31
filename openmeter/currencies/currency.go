package currencies

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CurrencyReference struct {
	Code             currencyx.Code `json:"code"`
	CustomCurrencyID *string        `json:"custom_currency_id,omitempty"`

	// resolved contains the Currency representation of the Code.
	resolved *Currency
}

const (
	currencyReferenceSerializationVersionV1 = "v1"
	currencyReferenceSerializationDelimiter = "|"
)

func NewCurrencyReference(code currencyx.Code) CurrencyReference {
	return CurrencyReference{Code: code}
}

// IsResolved reports whether the currency definition is available locally.
// Fiat definitions are resolved by code. Custom definitions are hydrated from
// the currency service or a persisted immutable snapshot.
func (r CurrencyReference) IsResolved() bool {
	if r.IsFiat() {
		return true
	}

	return r.CustomCurrencyID != nil &&
		r.resolved != nil
}

// IsCostBasisResolved reports whether currency resolution also expanded the
// custom currency's cost-basis history. A non-nil empty slice means the
// expansion completed but found no entries.
func (r CurrencyReference) IsCostBasisResolved() bool {
	if r.IsFiat() {
		return true
	}

	return r.IsResolved() && r.resolved.CostBasis != nil
}

func (r CurrencyReference) CustomCurrency() (*Currency, bool) {
	return r.resolved, r.resolved != nil
}

func (r CurrencyReference) GetCode() currencyx.Code {
	return r.Code
}

func (r CurrencyReference) IsFiat() bool {
	return r.Code.IsFiat()
}

func (r CurrencyReference) IsCustom() bool {
	return r.Code.IsCustom()
}

// Equal compares persisted reference identity and deliberately ignores the
// runtime-only resolved currency.
func (r CurrencyReference) Equal(other CurrencyReference) bool {
	return r.Code == other.Code && lo.FromPtr(r.CustomCurrencyID) == lo.FromPtr(other.CustomCurrencyID)
}

// IdentityKey returns the stable comparable identity used when values must be
// grouped without carrying the runtime-resolved currency definition.
func (r CurrencyReference) IdentityKey() string {
	if r.CustomCurrencyID != nil {
		return "CUSTOM:" + r.Code.String() + ":" + *r.CustomCurrencyID
	}

	return "FIAT:" + r.Code.String()
}

func (r CurrencyReference) Clone() CurrencyReference {
	if r.CustomCurrencyID != nil {
		r.CustomCurrencyID = lo.ToPtr(*r.CustomCurrencyID)
	}

	if r.resolved != nil {
		resolved := r.resolved.Clone()
		r.resolved = &resolved
	}

	return r
}

func (r CurrencyReference) String() string {
	if r.CustomCurrencyID != nil {
		return fmt.Sprintf("%s [type=%s id=%s]", r.Code, r.Code.Type(), *r.CustomCurrencyID)
	}

	return fmt.Sprintf("%s [type=%s]", r.Code, r.Code.Type())
}

// MarshalText returns the stable persisted representation used by ledger
// currency dimensions. Custom references must be hydrated so their immutable
// precision snapshot is retained without a later currency-service lookup.
func (r CurrencyReference) MarshalText() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	if r.IsFiat() {
		return []byte(r.Code), nil
	}

	if !r.IsResolved() {
		return nil, errors.New("custom currency reference must be resolved")
	}

	currency, _ := r.CustomCurrency()

	return []byte(strings.Join([]string{
		"custom",
		currencyReferenceSerializationVersionV1,
		r.Code.String(),
		*r.CustomCurrencyID,
		strconv.FormatUint(uint64(currency.Details().Precision), 10),
	}, currencyReferenceSerializationDelimiter)), nil
}

// MarshalTextPrefix returns the stable storage prefix for an unresolved
// custom-currency filter. Resolved references should use MarshalText for an
// exact match.
func (r CurrencyReference) MarshalTextPrefix() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	if r.IsFiat() {
		return []byte(r.Code), nil
	}

	segments := []string{
		"custom",
		currencyReferenceSerializationVersionV1,
		r.Code.String(),
	}
	if r.CustomCurrencyID != nil {
		segments = append(segments, *r.CustomCurrencyID)
	}

	return []byte(strings.Join(segments, currencyReferenceSerializationDelimiter) + currencyReferenceSerializationDelimiter), nil
}

// MarshalJSON preserves the public object representation after implementing
// encoding.TextMarshaler for the ledger currency dimension.
func (r CurrencyReference) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code             currencyx.Code `json:"code"`
		CustomCurrencyID *string        `json:"custom_currency_id,omitempty"`
	}{
		Code:             r.Code,
		CustomCurrencyID: r.CustomCurrencyID,
	})
}

func ParseCurrencyReference(value []byte) (CurrencyReference, error) {
	serialized := string(value)
	segments := strings.Split(serialized, currencyReferenceSerializationDelimiter)
	if len(segments) == 1 {
		reference := NewCurrencyReference(currencyx.Code(serialized))
		if err := reference.Validate(); err != nil {
			return CurrencyReference{}, err
		}
		if !reference.IsFiat() {
			return CurrencyReference{}, errors.New("custom currency reference snapshot is required")
		}

		return reference, nil
	}

	if len(segments) != 5 || segments[0] != "custom" {
		return CurrencyReference{}, fmt.Errorf("invalid currency reference %q", serialized)
	}
	if segments[1] != currencyReferenceSerializationVersionV1 {
		return CurrencyReference{}, fmt.Errorf("unsupported currency reference version %q", segments[1])
	}

	code := currencyx.Code(segments[2])
	if !code.IsCustom() {
		return CurrencyReference{}, fmt.Errorf("custom currency reference requires a custom currency code: %q", code)
	}
	if segments[3] == "" {
		return CurrencyReference{}, errors.New("custom currency id is required")
	}

	precision, err := strconv.ParseUint(segments[4], 10, 32)
	if err != nil {
		return CurrencyReference{}, fmt.Errorf("invalid custom currency precision %q: %w", segments[4], err)
	}

	resolved, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(code).
		WithName(code.String()).
		WithPrecision(uint32(precision)).
		Build()
	if err != nil {
		return CurrencyReference{}, fmt.Errorf("build custom currency snapshot: %w", err)
	}

	currency := Currency{
		NamespacedID: models.NamespacedID{ID: segments[3]},
		Currency:     resolved,
	}

	return currency.Reference(), nil
}

func (r CurrencyReference) Validate() error {
	var errs []error

	if err := r.Code.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("code: %w", err))
	}

	if r.IsFiat() && r.CustomCurrencyID != nil {
		errs = append(errs, errors.New("fiat currency cannot have a custom currency id"))
	}

	if r.resolved != nil {
		if err := r.resolved.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("resolved currency: %w", err))
		} else {
			if r.Code != r.resolved.GetCode() {
				errs = append(errs, fmt.Errorf("code mismatch between reference and resolved [reference.code=%s resolved.code=%s]", r.Code, r.resolved.GetCode()))
			}

			if r.IsCustom() {
				switch {
				case r.CustomCurrencyID == nil:
					errs = append(errs, errors.New("resolved custom currency id is required"))
				case *r.CustomCurrencyID != r.resolved.ID:
					errs = append(errs, fmt.Errorf("id mismatch between reference and resolved [reference.id=%s resolved.id=%s]", *r.CustomCurrencyID, r.resolved.ID))
				}
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r CurrencyReference) WithCurrency(currency *Currency) (CurrencyReference, error) {
	if currency == nil {
		return CurrencyReference{}, errors.New("currency is required")
	}

	if r.IsFiat() {
		return CurrencyReference{}, errors.New("fiat currency cannot have a custom currency")
	}

	if r.Code != currency.GetCode() {
		return CurrencyReference{}, fmt.Errorf("code mismatch between reference and currency [reference.code=%s resolved.code=%s]", r.Code, currency.GetCode())
	}

	if r.CustomCurrencyID != nil && currency.ID != *r.CustomCurrencyID {
		return CurrencyReference{}, fmt.Errorf("id mismatch between reference and currency [reference.id=%s resolved.id=%s]", *r.CustomCurrencyID, currency.ID)
	}

	if err := currency.Validate(); err != nil {
		return CurrencyReference{}, fmt.Errorf("invalid resolved currency: %w", err)
	}

	r.CustomCurrencyID = lo.EmptyableToPtr(currency.ID)

	r.resolved = currency

	if err := r.Validate(); err != nil {
		return CurrencyReference{}, err
	}

	return r, nil
}

type Currency struct {
	models.ManagedModel
	models.NamespacedID
	currencyx.Currency

	// CostBasis is included only if the Currency is expanded.
	CostBasis *[]CostBasis `json:"-"`
}

func (c Currency) Reference() CurrencyReference {
	ref := CurrencyReference{
		Code:     c.GetCode(),
		resolved: &c,
	}
	if c.IsCustom() {
		ref.CustomCurrencyID = lo.ToPtr(c.ID)
	}

	return ref
}

func NewFiatCurrency(code currencyx.Code) (Currency, error) {
	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(code).
		Build()
	if err != nil {
		return Currency{}, fmt.Errorf("build fiat currency: %w", err)
	}

	return Currency{Currency: currency}, nil
}

func (c Currency) Validate() error {
	var errs []error

	if c.Currency == nil {
		errs = append(errs, errors.New("currency is required"))
	} else {
		if err := c.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}

		if c.Currency.Type() == currencyx.CurrencyTypeCustom && c.ID == "" {
			errs = append(errs, errors.New("managed custom currency ID is required"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GetCode returns the resolved currency code, or an empty code when the currency is missing.
func (c Currency) GetCode() currencyx.Code {
	if c.Currency == nil {
		return ""
	}

	return c.Currency.Details().Code
}

func (c Currency) IsFiat() bool {
	return c.Currency != nil && c.Currency.Type() == currencyx.CurrencyTypeFiat
}

func (c Currency) IsCustom() bool {
	return c.Currency != nil && c.Currency.Type() == currencyx.CurrencyTypeCustom
}

func (c Currency) Clone() Currency {
	if c.CostBasis != nil {
		c.CostBasis = lo.ToPtr(slices.Clone(*c.CostBasis))
	}

	return c
}

// Identity uniquely identifies a currency by its fiat code or custom currency ID.
func (c Currency) Identity() (string, error) {
	if c.IsFiat() {
		return fmt.Sprintf("FIAT:%s", c.GetCode()), nil
	}

	if c.IsCustom() {
		return fmt.Sprintf("CUSTOM:%s:%s", c.Namespace, c.ID), nil
	}

	return "", fmt.Errorf("currency is not fiat or custom")
}

var (
	_ models.Validator   = Currency{}
	_ currencyx.Currency = (*Currency)(nil)
)

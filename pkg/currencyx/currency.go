package currencyx

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"

	"github.com/openmeterio/openmeter/pkg/models"
)

type CurrencyType string

const (
	CurrencyTypeFiat   CurrencyType = "fiat"
	CurrencyTypeCustom CurrencyType = "custom"
)

func (t CurrencyType) Validate() error {
	var errs []error

	switch t {
	case CurrencyTypeFiat, CurrencyTypeCustom:
	default:
		errs = append(errs, fmt.Errorf("invalid currency type: %s", t))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CurrencyFormatter interface {
	FormatAmount(amount alpacadecimal.Decimal) string
}

type CurrencyCalculator interface {
	RoundToPrecision(amount alpacadecimal.Decimal) alpacadecimal.Decimal
	IsRoundedToPrecision(amount alpacadecimal.Decimal) bool

	RoundUp(amount alpacadecimal.Decimal) alpacadecimal.Decimal
	RoundDown(amount alpacadecimal.Decimal) alpacadecimal.Decimal

	Unit() alpacadecimal.Decimal
}

type Currency interface {
	models.Validator
	CurrencyCalculator
	CurrencyFormatter

	Type() CurrencyType
	Details() CurrencyDetails

	AsFiat() (*FiatCurrency, error)
	AsCustom() (*CustomCurrency, error)

	Definition() *currency.Def
}

type CurrencyDetails struct {
	Code               Code   `json:"code"`
	Name               string `json:"name"`
	Symbol             string `json:"symbol,omitempty"`
	Precision          uint32 `json:"precision"`
	DecimalMark        string `json:"decimal_mark,omitempty"`
	ThousandsSeparator string `json:"thousands_separator,omitempty"`
}

// formatCurrencyAmount renders amount using def's formatting rules, shared by every
// type implementing Currency.
func formatCurrencyAmount(def *currency.Def, amount alpacadecimal.Decimal) string {
	abs := amount.Abs()

	var formatted string

	if abs.IsInteger() {
		formatted = def.FormatAmount(num.MakeAmount(abs.IntPart(), 0))
	} else {
		numAmount := num.MakeAmount(abs.CoefficientInt64(), uint32(-abs.Exponent()))
		formatted = def.FormatAmount(def.Rescale(numAmount))
	}

	if amount.IsNegative() {
		return "-" + formatted
	}

	return formatted
}

// cloneDef returns a deep copy of def, so a caller can customize the copy (e.g.
// override formatting fields) without mutating the shared, package-level
// definitions returned by currency.Get. AlternateSymbols is the only reference
// type on currency.Def, so it's the only field that needs an explicit clone.
func cloneDef(def *currency.Def) *currency.Def {
	if def == nil {
		return nil
	}

	clone := *def
	clone.AlternateSymbols = slices.Clone(def.AlternateSymbols)

	return &clone
}

var (
	_ models.Validator                 = (*FiatCurrency)(nil)
	_ models.CustomValidator[Currency] = (*FiatCurrency)(nil)
	_ Currency                         = (*FiatCurrency)(nil)
)

type FiatCurrency struct {
	def *currency.Def
}

func (f *FiatCurrency) Definition() *currency.Def {
	return cloneDef(f.def)
}

func (f *FiatCurrency) FormatAmount(amount alpacadecimal.Decimal) string {
	return formatCurrencyAmount(f.def, amount)
}

func (f *FiatCurrency) Unit() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(1).Shift(-int32(f.def.Subunits))
}

func (f *FiatCurrency) RoundUp(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	return amount.RoundUp(int32(f.def.Subunits))
}

func (f *FiatCurrency) RoundDown(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	return amount.RoundDown(int32(f.def.Subunits))
}

func (f *FiatCurrency) IsRoundedToPrecision(amount alpacadecimal.Decimal) bool {
	return amount.Equal(f.RoundToPrecision(amount))
}

func (f *FiatCurrency) RoundToPrecision(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	return amount.Round(int32(f.def.Subunits))
}

func (f *FiatCurrency) Type() CurrencyType {
	return CurrencyTypeFiat
}

func (f *FiatCurrency) Details() CurrencyDetails {
	return CurrencyDetails{
		Code:               Code(f.def.ISOCode),
		Name:               f.def.Name,
		Precision:          f.def.Subunits,
		Symbol:             f.def.Symbol,
		DecimalMark:        f.def.DecimalMark,
		ThousandsSeparator: f.def.ThousandsSeparator,
	}
}

func (f *FiatCurrency) AsFiat() (*FiatCurrency, error) {
	return &FiatCurrency{
		def: cloneDef(f.def),
	}, nil
}

func (f *FiatCurrency) AsCustom() (*CustomCurrency, error) {
	return nil, fmt.Errorf("cannot convert fiat to custom currency")
}

func (f *FiatCurrency) ValidateWith(v ...models.ValidatorFunc[Currency]) error {
	return models.Validate[Currency](f, v...)
}

func (f *FiatCurrency) Validate() error {
	if f == nil {
		return errors.New("fiat currency is not initialized")
	}

	var errs []error

	if f.def == nil || f.def.ISOCode == "" {
		errs = append(errs, errors.New("invalid fiat currency: empty code"))
	}

	if f.def != nil {
		if f.def.Name == "" {
			errs = append(errs, errors.New("invalid fiat currency: empty name"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func newFiatCurrency(code string) (Currency, error) {
	definition := currency.Get(currency.Code(code))
	if definition == nil || definition.ISONumeric == "" {
		return nil, fmt.Errorf("invalid fiat currency code: %s", code)
	}

	if definition.Subunits > math.MaxInt32 {
		return nil, fmt.Errorf("value %d overflows int32", definition.Subunits)
	}

	return &FiatCurrency{
		def: definition,
	}, nil
}

var (
	_ models.Validator                 = (*CustomCurrency)(nil)
	_ models.CustomValidator[Currency] = (*CustomCurrency)(nil)
	_ Currency                         = (*CustomCurrency)(nil)
)

type CustomCurrency struct {
	def *currency.Def
}

func (c *CustomCurrency) Definition() *currency.Def {
	return cloneDef(c.def)
}

func (c *CustomCurrency) FormatAmount(amount alpacadecimal.Decimal) string {
	return formatCurrencyAmount(c.def, amount)
}

func (c *CustomCurrency) Unit() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(1).Shift(-int32(c.def.Subunits))
}

func (c *CustomCurrency) RoundUp(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	return amount.RoundUp(int32(c.def.Subunits))
}

func (c *CustomCurrency) RoundDown(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	return amount.RoundDown(int32(c.def.Subunits))
}

func (c *CustomCurrency) IsRoundedToPrecision(amount alpacadecimal.Decimal) bool {
	return amount.Equal(c.RoundToPrecision(amount))
}

func (c *CustomCurrency) RoundToPrecision(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	return amount.Round(int32(c.def.Subunits))
}

func (c *CustomCurrency) Type() CurrencyType {
	return CurrencyTypeCustom
}

func (c *CustomCurrency) Details() CurrencyDetails {
	return CurrencyDetails{
		Code:               Code(c.def.ISOCode),
		Name:               c.def.Name,
		Symbol:             c.def.Symbol,
		Precision:          c.def.Subunits,
		DecimalMark:        c.def.DecimalMark,
		ThousandsSeparator: c.def.ThousandsSeparator,
	}
}

func (c *CustomCurrency) AsFiat() (*FiatCurrency, error) {
	return nil, fmt.Errorf("cannot convert custom to fiat currency")
}

func (c *CustomCurrency) AsCustom() (*CustomCurrency, error) {
	return &CustomCurrency{
		def: cloneDef(c.def),
	}, nil
}

func (c *CustomCurrency) ValidateWith(v ...models.ValidatorFunc[Currency]) error {
	return models.Validate[Currency](c, v...)
}

const (
	CustomCurrencyCodeMinLength = 4
	CustomCurrencyCodeMaxLength = 24

	CustomCurrencyMaxPrecision uint32 = 12
)

func (c *CustomCurrency) Validate() error {
	if c == nil {
		return errors.New("custom currency is not initialized")
	}

	var errs []error

	if c.def == nil {
		errs = append(errs, errors.New("currency is not initialized"))
	}

	if c.def != nil {
		if c.def.ISOCode == "" {
			errs = append(errs, errors.New("code is required"))
		}

		if len(c.def.ISOCode) != len(strings.TrimSpace(c.def.ISOCode.String())) {
			errs = append(errs, fmt.Errorf("invalid currency code: cannot contain leading or trailing spaces: %s", c.def.ISOCode))
		}

		if strings.Contains(c.def.ISOCode.String(), "|") {
			errs = append(errs, fmt.Errorf("invalid currency code: cannot contain route delimiter: %s", c.def.ISOCode))
		}

		if fiatDef := currency.Get(c.def.ISOCode); fiatDef != nil {
			errs = append(errs, fmt.Errorf("currency code %s is a fiat currency", c.def.ISOCode))
		}

		if cl := len(c.def.ISOCode); cl < CustomCurrencyCodeMinLength || cl > CustomCurrencyCodeMaxLength {
			errs = append(errs, fmt.Errorf("invalid currency code: it must be between %d and %d characters", CustomCurrencyCodeMinLength, CustomCurrencyCodeMaxLength))
		}

		if c.def.Name == "" {
			errs = append(errs, errors.New("name is required"))
		}

		if c.def.Subunits > CustomCurrencyMaxPrecision {
			errs = append(errs, fmt.Errorf("invalid precision: it must be between 0 and %d", CustomCurrencyMaxPrecision))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func newCustomCurrency(def *currency.Def) (Currency, error) {
	cc := CustomCurrency{
		def: def,
	}
	if err := cc.Validate(); err != nil {
		return nil, err
	}

	return &cc, nil
}

type CurrencyBuilder struct {
	t CurrencyType
	d CurrencyDetails
}

func (b *CurrencyBuilder) WithCode(code Code) *CurrencyBuilder {
	b.d.Code = code

	return b
}

func (b *CurrencyBuilder) WithName(name string) *CurrencyBuilder {
	b.d.Name = name

	return b
}

func (b *CurrencyBuilder) WithSymbol(symbol string) *CurrencyBuilder {
	b.d.Symbol = symbol

	return b
}

func (b *CurrencyBuilder) WithPrecision(precision uint32) *CurrencyBuilder {
	b.d.Precision = precision

	return b
}

func (b *CurrencyBuilder) WithDecimalMark(decimalMark string) *CurrencyBuilder {
	b.d.DecimalMark = decimalMark

	return b
}

func (b *CurrencyBuilder) WithThousandsSeparator(thousandsSeparator string) *CurrencyBuilder {
	b.d.ThousandsSeparator = thousandsSeparator

	return b
}

func (b *CurrencyBuilder) Build() (Currency, error) {
	switch b.t {
	case CurrencyTypeFiat:
		return newFiatCurrency(b.d.Code.String())
	case CurrencyTypeCustom:
		return newCustomCurrency(&currency.Def{
			ISOCode:            currency.Code(b.d.Code.String()),
			Name:               b.d.Name,
			Symbol:             b.d.Symbol,
			Subunits:           b.d.Precision,
			Template:           currency.DefaultCurrencyTemplate,
			DecimalMark:        b.d.DecimalMark,
			ThousandsSeparator: b.d.ThousandsSeparator,
			NumeralSystem:      num.NumeralWestern,
		})
	default:
		return nil, fmt.Errorf("invalid currency type: %s", b.t)
	}
}

func NewCurrencyBuilder(currencyType CurrencyType) *CurrencyBuilder {
	return &CurrencyBuilder{
		t: currencyType,
		d: CurrencyDetails{
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	}
}

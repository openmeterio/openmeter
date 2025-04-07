package billing

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

const (
	// LineMaximumSpendReferenceID is a discount applied due to maximum spend.
	LineMaximumSpendReferenceID = "line_maximum_spend"
)

type LineDiscountReason string

const (
	LineDiscountReasonMaximumSpend     LineDiscountReason = "maximum_spend"
	LineDiscountReasonRatecardDiscount LineDiscountReason = "ratecard_discount"
)

func (LineDiscountReason) Values() []string {
	return []string{
		string(LineDiscountReasonMaximumSpend),
		string(LineDiscountReasonRatecardDiscount),
	}
}

type LineSetDiscountMangedFields struct {
	ID                  string `json:"id"`
	models.ManagedModel `json:",inline"`
}

func (i LineSetDiscountMangedFields) Validate() error {
	// We are ignoring the managed model validation as quite often we are relying on
	// idempotent calculations instead.
	return nil
}

func (i LineSetDiscountMangedFields) Equal(other LineSetDiscountMangedFields) bool {
	if i.ID != other.ID {
		return false
	}

	return i.ManagedModel.Equal(other.ManagedModel)
}

func (i LineSetDiscountMangedFields) SetDeletedAt(deletedAt time.Time) LineSetDiscountMangedFields {
	i.DeletedAt = &deletedAt
	return i
}

type LineDiscountBase struct {
	Description            *string            `json:"description,omitempty"`
	ChildUniqueReferenceID *string            `json:"childUniqueReferenceId,omitempty"`
	ExternalIDs            LineExternalIDs    `json:"externalIDs,omitempty"`
	Reason                 LineDiscountReason `json:"reason"`

	SourceDiscount *productcatalog.Discount `json:"rateCardDiscount,omitempty"`
}

func (i LineDiscountBase) Validate() error {
	var errs []error

	if i.Reason == "" {
		errs = append(errs, errors.New("reason is required"))
	}

	if i.SourceDiscount != nil {
		if err := i.SourceDiscount.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("sourceDiscount: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (i LineDiscountBase) Equal(other LineDiscountBase) bool {
	return reflect.DeepEqual(i, other)
}

func (i LineDiscountBase) Clone() LineDiscountBase {
	return i
}

type LineDiscountBaseManaged struct {
	LineSetDiscountMangedFields `json:",inline"`
	LineDiscountBase            `json:",inline"`
}

type AmountLineDiscount struct {
	LineDiscountBase `json:",inline"`

	Amount alpacadecimal.Decimal `json:"amount"`

	// RoundingAmount is a correction value, to ensure that if multiple discounts are applied,
	// then sum of discount amounts equals the total * sum(discount percentages).
	RoundingAmount alpacadecimal.Decimal `json:"rounding"`
}

func (i AmountLineDiscount) Validate() error {
	var errs []error

	if err := i.LineDiscountBase.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Amount.IsNegative() {
		errs = append(errs, errors.New("amount should be positive or zero"))
	}

	return errors.Join(errs...)
}

func (i AmountLineDiscount) Equal(other AmountLineDiscount) bool {
	if !i.LineDiscountBase.Equal(other.LineDiscountBase) {
		return false
	}

	if !i.Amount.Equal(other.Amount) {
		return false
	}

	if !i.RoundingAmount.Equal(other.RoundingAmount) {
		return false
	}

	return true
}

func (i AmountLineDiscount) Clone() AmountLineDiscount {
	return AmountLineDiscount{
		LineDiscountBase: i.LineDiscountBase.Clone(),
		Amount:           i.Amount,
		RoundingAmount:   i.RoundingAmount,
	}
}

type AmountLineDiscountManaged struct {
	LineSetDiscountMangedFields `json:",inline"`
	AmountLineDiscount          `json:",inline"`
}

func (i AmountLineDiscountManaged) Validate() error {
	var errs []error

	if err := i.LineSetDiscountMangedFields.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.AmountLineDiscount.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (i AmountLineDiscountManaged) Equal(other AmountLineDiscountManaged) bool {
	if !i.LineSetDiscountMangedFields.Equal(other.LineSetDiscountMangedFields) {
		return false
	}

	if !i.AmountLineDiscount.Equal(other.AmountLineDiscount) {
		return false
	}

	return true
}

func (i AmountLineDiscountManaged) Clone() AmountLineDiscountManaged {
	return AmountLineDiscountManaged{
		LineSetDiscountMangedFields: i.LineSetDiscountMangedFields,
		AmountLineDiscount:          i.AmountLineDiscount.Clone(),
	}
}

type UsageLineDiscount struct {
	LineDiscountBase `json:",inline"`

	Quantity              alpacadecimal.Decimal  `json:"quantity"`
	PreLinePeriodQuantity *alpacadecimal.Decimal `json:"preLinePeriodQuantity"`
}

func (i UsageLineDiscount) Validate() error {
	var errs []error

	if err := i.LineDiscountBase.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Quantity.IsNegative() {
		errs = append(errs, errors.New("quantity should be positive or zero"))
	}

	if i.PreLinePeriodQuantity != nil && i.PreLinePeriodQuantity.IsNegative() {
		errs = append(errs, errors.New("preLinePeriodQuantity should be positive or zero"))
	}

	return errors.Join(errs...)
}

func (i UsageLineDiscount) Equal(other UsageLineDiscount) bool {
	if !i.LineDiscountBase.Equal(other.LineDiscountBase) {
		return false
	}

	if !i.Quantity.Equal(other.Quantity) {
		return false
	}

	if !equal.PtrEqual(i.PreLinePeriodQuantity, other.PreLinePeriodQuantity) {
		return false
	}

	return true
}

func (i UsageLineDiscount) Clone() UsageLineDiscount {
	return UsageLineDiscount{
		LineDiscountBase:      i.LineDiscountBase.Clone(),
		Quantity:              i.Quantity,
		PreLinePeriodQuantity: i.PreLinePeriodQuantity,
	}
}

type UsageLineDiscountManaged struct {
	LineSetDiscountMangedFields `json:",inline"`
	UsageLineDiscount           `json:",inline"`
}

func (i UsageLineDiscountManaged) Validate() error {
	var errs []error

	if err := i.LineSetDiscountMangedFields.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.UsageLineDiscount.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (i UsageLineDiscountManaged) Equal(other UsageLineDiscountManaged) bool {
	if !i.LineSetDiscountMangedFields.Equal(other.LineSetDiscountMangedFields) {
		return false
	}

	if !i.UsageLineDiscount.Equal(other.UsageLineDiscount) {
		return false
	}

	return true
}

func (i UsageLineDiscountManaged) Clone() UsageLineDiscountManaged {
	return UsageLineDiscountManaged{
		LineSetDiscountMangedFields: i.LineSetDiscountMangedFields,
		UsageLineDiscount:           i.UsageLineDiscount.Clone(),
	}
}

type LineDiscountType string

const (
	LineDiscountTypeAmount LineDiscountType = "amount"
	LineDiscountTypeUsage  LineDiscountType = "usage"
)

func (LineDiscountType) Values() []string {
	return []string{
		string(LineDiscountTypeAmount),
		string(LineDiscountTypeUsage),
	}
}

type LineDiscountMutator func(discount LineDiscount) (LineDiscount, error)

type LineDiscount interface {
	models.Equaler[LineDiscount]
	models.Validator
	models.Clonable[LineDiscount]

	Type() LineDiscountType

	// ContentEqual checks if the discount has the same contents as the other discount, ignoring managed fields.
	ContentsEqual(other LineDiscount) bool

	// AsAmount returns the amount discount if the discount is an amount discount, if no error is returned
	// the pointer is guaranteed to be non-nil. A pointer is used as invocing relies on in-place manipulation
	// of the invoice.
	AsAmount() (AmountLineDiscountManaged, error)

	// AsUsage returns the usage discount if the discount is a usage discount, if no error is returned
	// the pointer is guaranteed to be non-nil. A pointer is used as invocing relies on in-place manipulation
	// of the invoice.
	AsUsage() (UsageLineDiscountManaged, error)

	// AsDiscountBase returns the discount base if the discount is a discount base, if no error is returned
	// the pointer is guaranteed to be non-nil. A pointer is used as invocing relies on in-place manipulation
	// of the invoice.
	AsDiscountBase() (LineDiscountBaseManaged, error)

	// Mutate mutates the discount and returns the modified discount.
	Mutate(...LineDiscountMutator) (LineDiscount, error)

	// Common field accessors
	GetID() string
	GetChildUniqueReferenceID() *string
	GetManagedFields() LineSetDiscountMangedFields

	// Internal field accessors (they don't have nil or type checks)
	amountManaged() *AmountLineDiscountManaged
	usageManaged() *UsageLineDiscountManaged
}

type lineDiscount struct {
	t LineDiscountType

	amount *AmountLineDiscountManaged
	usage  *UsageLineDiscountManaged
}

func NewLineDiscountFrom[T AmountLineDiscountManaged | AmountLineDiscount | UsageLineDiscountManaged | UsageLineDiscount](discount T) LineDiscount {
	switch any(discount).(type) {
	// Allow provisioning an AmountLineDiscountManaged without the managed fields: given we are calculating the expected state then merge
	// with the DB state, this is the most common case.
	case AmountLineDiscount:
		amount := any(discount).(AmountLineDiscount)

		return lineDiscount{t: LineDiscountTypeAmount, amount: &AmountLineDiscountManaged{
			AmountLineDiscount: amount,
		}}
	case AmountLineDiscountManaged:
		amount := any(discount).(AmountLineDiscountManaged)

		return lineDiscount{t: LineDiscountTypeAmount, amount: &amount}
	case UsageLineDiscountManaged:
		usage := any(discount).(UsageLineDiscountManaged)

		return lineDiscount{t: LineDiscountTypeUsage, usage: &usage}
	case UsageLineDiscount:
		usage := any(discount).(UsageLineDiscount)

		return lineDiscount{t: LineDiscountTypeUsage, usage: &UsageLineDiscountManaged{
			UsageLineDiscount: usage,
		}}
	}

	return lineDiscount{}
}

func (i lineDiscount) Type() LineDiscountType {
	return i.t
}

func (i lineDiscount) AsAmount() (AmountLineDiscountManaged, error) {
	empty := AmountLineDiscountManaged{}

	if i.t != LineDiscountTypeAmount {
		return empty, errors.New("not an amount discount")
	}

	if i.amount == nil {
		return empty, errors.New("amount discount is nil")
	}

	return *i.amount, nil
}

func (i lineDiscount) AsUsage() (UsageLineDiscountManaged, error) {
	empty := UsageLineDiscountManaged{}

	if i.t != LineDiscountTypeUsage {
		return empty, errors.New("not a usage discount")
	}

	if i.usage == nil {
		return empty, errors.New("usage discount is nil")
	}

	return *i.usage, nil
}

func (i lineDiscount) amountManaged() *AmountLineDiscountManaged {
	return i.amount
}

func (i lineDiscount) usageManaged() *UsageLineDiscountManaged {
	return i.usage
}

func (i lineDiscount) AsDiscountBase() (LineDiscountBaseManaged, error) {
	empty := LineDiscountBaseManaged{}

	switch i.t {
	case LineDiscountTypeAmount:
		if i.amount == nil {
			return empty, errors.New("amount discount is nil")
		}

		return LineDiscountBaseManaged{
			LineSetDiscountMangedFields: i.amount.LineSetDiscountMangedFields,
			LineDiscountBase:            i.amount.LineDiscountBase,
		}, nil
	case LineDiscountTypeUsage:
		if i.usage == nil {
			return empty, errors.New("usage discount is nil")
		}

		return LineDiscountBaseManaged{
			LineSetDiscountMangedFields: i.usage.LineSetDiscountMangedFields,
			LineDiscountBase:            i.usage.LineDiscountBase,
		}, nil
	default:
		return empty, errors.New("unknown discount type")
	}
}

func (i lineDiscount) Equal(other LineDiscount) bool {
	if i.t != other.Type() {
		return false
	}

	switch i.t {
	case LineDiscountTypeAmount:
		return equal.PtrEqual(i.amount, other.amountManaged())
	case LineDiscountTypeUsage:
		return equal.PtrEqual(i.usage, other.usageManaged())
	default:
		return false
	}
}

func (i lineDiscount) ContentsEqual(other LineDiscount) bool {
	if i.t != other.Type() {
		return false
	}

	switch i.t {
	case LineDiscountTypeAmount:
		// Invalid state, should never happen
		if i.amount == nil || other.amountManaged() == nil {
			return false
		}

		return i.amount.AmountLineDiscount.Equal(other.amountManaged().AmountLineDiscount)
	case LineDiscountTypeUsage:
		// Invalid state, should never happen
		if i.usage == nil || other.usageManaged() == nil {
			return false
		}

		return i.usage.UsageLineDiscount.Equal(other.usageManaged().UsageLineDiscount)
	default:
		return false
	}
}

func (i lineDiscount) Mutate(mutators ...LineDiscountMutator) (LineDiscount, error) {
	out := LineDiscount(i)
	for _, mutator := range mutators {
		newValue, err := mutator(out)
		if err != nil {
			return out, err
		}

		out = newValue
	}

	return out, nil
}

func (i lineDiscount) GetID() string {
	return i.GetManagedFields().ID
}

func (i lineDiscount) GetManagedFields() LineSetDiscountMangedFields {
	switch i.t {
	case LineDiscountTypeAmount:
		return i.amount.LineSetDiscountMangedFields
	case LineDiscountTypeUsage:
		return i.usage.LineSetDiscountMangedFields
	default:
		return LineSetDiscountMangedFields{}
	}
}

func (i lineDiscount) GetChildUniqueReferenceID() *string {
	switch i.t {
	case LineDiscountTypeAmount:
		return i.amount.LineDiscountBase.ChildUniqueReferenceID
	case LineDiscountTypeUsage:
		return i.usage.LineDiscountBase.ChildUniqueReferenceID
	default:
		return nil
	}
}

func (i lineDiscount) Validate() error {
	switch i.t {
	case LineDiscountTypeAmount:
		return i.amount.Validate()
	case LineDiscountTypeUsage:
		return i.usage.Validate()
	default:
		return errors.New("unknown discount type")
	}
}

func (i lineDiscount) Clone() LineDiscount {
	switch i.t {
	case LineDiscountTypeAmount:
		return &lineDiscount{
			t:      i.t,
			amount: lo.ToPtr(i.amount.Clone()),
		}
	case LineDiscountTypeUsage:
		return &lineDiscount{t: i.t, usage: lo.ToPtr(i.usage.Clone())}
	default:
		return &lineDiscount{}
	}
}

// LineDiscount mutators

func SetDiscountInvoicingExternalID(externalID string) LineDiscountMutator {
	return func(discount LineDiscount) (LineDiscount, error) {
		var out LineDiscount
		switch discount.Type() {
		case LineDiscountTypeAmount:
			amount, err := discount.AsAmount()
			if err != nil {
				return discount, err
			}

			amount.ExternalIDs.Invoicing = externalID
			out = NewLineDiscountFrom(amount)
		case LineDiscountTypeUsage:
			usage, err := discount.AsUsage()
			if err != nil {
				return discount, err
			}

			usage.ExternalIDs.Invoicing = externalID
			out = NewLineDiscountFrom(usage)
		default:
			return discount, fmt.Errorf("unknown discount type: %s", discount.Type())
		}

		return out, nil
	}
}

func SetDiscountMangedFields(managedFields LineSetDiscountMangedFields) LineDiscountMutator {
	return func(discount LineDiscount) (LineDiscount, error) {
		var out LineDiscount
		switch discount.Type() {
		case LineDiscountTypeAmount:
			amount, err := discount.AsAmount()
			if err != nil {
				return discount, err
			}

			amount.LineSetDiscountMangedFields = managedFields
			out = NewLineDiscountFrom(amount)
		case LineDiscountTypeUsage:
			usage, err := discount.AsUsage()
			if err != nil {
				return discount, err
			}

			usage.LineSetDiscountMangedFields = managedFields
			out = NewLineDiscountFrom(usage)
		default:
			return discount, fmt.Errorf("unknown discount type: %s", discount.Type())
		}

		return out, nil
	}
}

func EditAmountDiscount(f func(AmountLineDiscountManaged) (AmountLineDiscountManaged, error)) LineDiscountMutator {
	return func(discount LineDiscount) (LineDiscount, error) {
		if discount.Type() != LineDiscountTypeAmount {
			return discount, fmt.Errorf("cannot edit non-amount discount: %s", discount.Type())
		}

		amount, err := discount.AsAmount()
		if err != nil {
			return discount, err
		}

		amount, err = f(amount)
		if err != nil {
			return discount, err
		}

		return NewLineDiscountFrom(amount), nil
	}
}

func MarkDiscountDeleted(deletedAt time.Time) LineDiscountMutator {
	return func(discount LineDiscount) (LineDiscount, error) {
		managedFields := discount.GetManagedFields()

		managedFields.DeletedAt = &deletedAt
		return SetDiscountMangedFields(managedFields)(discount)
	}
}

// LineDiscounts is a list of line discounts.
type LineDiscounts []LineDiscount

func NewLineDiscounts(discounts ...LineDiscount) LineDiscounts {
	return LineDiscounts(discounts)
}

func (d LineDiscounts) Clone() LineDiscounts {
	return LineDiscounts(lo.Map(d, func(discount LineDiscount, _ int) LineDiscount {
		return discount.Clone()
	}))
}

// ReuseIDsFrom reuses the IDs of the existing discounts by child unique reference ID.
// This is used to prevent creation of new IDs/rows in the database when doing an idempotent reconciliation.
func (d LineDiscounts) ReuseIDsFrom(existingItems []LineDiscount) (LineDiscounts, error) {
	existingItemsByUniqueReference := lo.GroupBy(
		lo.Filter(existingItems, func(item LineDiscount, _ int) bool {
			return item.GetChildUniqueReferenceID() != nil
		}),
		func(item LineDiscount) string {
			return *item.GetChildUniqueReferenceID()
		},
	)

	discountsWithIDReuse, err := slicesx.MapWithErr(d, func(discount LineDiscount) (LineDiscount, error) {
		childUniqueReferenceID := discount.GetChildUniqueReferenceID()

		// We should not reuse the ID if they are for a different child unique reference ID
		if childUniqueReferenceID == nil {
			return discount, nil
		}

		existingItems, ok := existingItemsByUniqueReference[*childUniqueReferenceID]
		if !ok {
			// We did not find any existing items for this child unique reference ID,
			// let's create a new entry in the DB.
			return discount, nil
		}

		existingManagedFields := existingItems[0].GetManagedFields()

		return discount.Mutate(SetDiscountMangedFields(LineSetDiscountMangedFields{
			ID: existingManagedFields.ID,
			ManagedModel: models.ManagedModel{
				CreatedAt: existingManagedFields.CreatedAt,
				// UpdatedAt is updated by the adapter layer
				// DeletedAt should not be set, to ensure that we are not carrying over soft-deletion flags
			},
		}))
	})
	if err != nil {
		return nil, err
	}

	return slicesx.EmptyAsNil(discountsWithIDReuse), nil
}

func (d LineDiscounts) SumAmount(currency currencyx.Calculator) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, amount := range d.GetAmountDiscounts() {
		sum = sum.Add(amount.Amount).Add(amount.RoundingAmount)
	}

	return sum
}

func (d LineDiscounts) Validate() error {
	var errs []error

	for _, discount := range d {
		if err := discount.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// GetAmountDiscounts returns all amount discounts in the list, the slice's items are never
// nil.
func (d LineDiscounts) GetAmountDiscounts() []AmountLineDiscountManaged {
	amountDiscounts := []AmountLineDiscountManaged{}

	for _, discount := range d {
		if discount.Type() != LineDiscountTypeAmount {
			continue
		}

		if amount, err := discount.AsAmount(); err == nil {
			amountDiscounts = append(amountDiscounts, amount)
		}
	}

	return amountDiscounts
}

// GetAmountDiscountsByID returns a map of amount discounts by their ID, the map's items are never
// nil. Items with an empty ID are not included in the map.
func (d LineDiscounts) GetAmountDiscountsByID() (map[string]AmountLineDiscountManaged, error) {
	amountDiscountsByID, unique := slicesx.UniqueGroupBy(
		lo.Filter(d.GetAmountDiscounts(), func(discount AmountLineDiscountManaged, _ int) bool {
			return discount.ID != ""
		}),
		func(discount AmountLineDiscountManaged) string {
			return discount.ID
		},
	)

	if !unique {
		return nil, errors.New("amount discounts are not unique")
	}

	return amountDiscountsByID, nil
}

// DiscountsByID returns a map of discounts by their ID, the map's items are never
// nil. Items with an empty ID are not included in the map.
func (d LineDiscounts) DiscountsByID() (map[string]LineDiscount, error) {
	discountsByID, unique := slicesx.UniqueGroupBy(
		lo.Filter(d, func(discount LineDiscount, _ int) bool {
			return discount.GetID() != ""
		}),
		func(discount LineDiscount) string {
			return discount.GetID()
		},
	)

	if !unique {
		return nil, errors.New("discountIDs are not unique")
	}

	return discountsByID, nil
}

func (d LineDiscounts) GetDiscountByChildUniqueReferenceID(childUniqueReferenceID string) (LineDiscount, bool) {
	for _, discount := range d {
		if discount.GetChildUniqueReferenceID() != nil && *discount.GetChildUniqueReferenceID() == childUniqueReferenceID {
			return discount, true
		}
	}
	return nil, false
}

// Mutate mutates the discounts in the list by applying the mutators in order, then
// returns the mutated discounts.
func (d LineDiscounts) Mutate(mutators ...LineDiscountMutator) (LineDiscounts, error) {
	return slicesx.MapWithErr(d, func(discount LineDiscount) (LineDiscount, error) {
		return discount.Mutate(mutators...)
	})
}

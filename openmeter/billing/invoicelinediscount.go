package billing

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

const (
	// LineMaximumSpendReferenceID is a discount applied due to maximum spend.
	LineMaximumSpendReferenceID = "line_maximum_spend"
)

type LineDiscountBase struct {
	Description            *string         `json:"description,omitempty"`
	ChildUniqueReferenceID *string         `json:"childUniqueReferenceId,omitempty"`
	ExternalIDs            LineExternalIDs `json:"externalIDs,omitempty"`
	Reason                 DiscountReason  `json:"reason,omitempty"`
}

func (i LineDiscountBase) Validate() error {
	var errs []error

	if err := i.Reason.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (i LineDiscountBase) Equal(other LineDiscountBase) bool {
	return deriveEqualLineDiscountBase(&i, &other)
}

func (i LineDiscountBase) Clone() LineDiscountBase {
	return i
}

func (i LineDiscountBase) GetChildUniqueReferenceID() *string {
	return i.ChildUniqueReferenceID
}

func (i LineDiscountBase) GetDescription() *string {
	return i.Description
}

type LineDiscountBaseManaged struct {
	models.ManagedModelWithID `json:",inline"`
	LineDiscountBase          `json:",inline"`
}

type AmountLineDiscount struct {
	LineDiscountBase `json:",inline"`

	Amount alpacadecimal.Decimal `json:"amount"`

	// RoundingAmount is a correction value, to ensure that if multiple discounts are applied,
	// then sum of discount amounts equals the total * sum(discount percentages).
	RoundingAmount alpacadecimal.Decimal `json:"roundingAmount"`
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
	return deriveEqualAmountLineDiscount(&i, &other)
}

func (i AmountLineDiscount) Clone() AmountLineDiscount {
	return AmountLineDiscount{
		LineDiscountBase: i.LineDiscountBase.Clone(),
		Amount:           i.Amount,
		RoundingAmount:   i.RoundingAmount,
	}
}

type AmountLineDiscountManaged struct {
	models.ManagedModelWithID `json:",inline"`
	AmountLineDiscount        `json:",inline"`
}

func (i AmountLineDiscountManaged) Validate() error {
	var errs []error

	if err := i.AmountLineDiscount.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (i AmountLineDiscountManaged) Equal(other AmountLineDiscountManaged) bool {
	return deriveEqualAmountLineDiscountManaged(&i, &other)
}

func (i AmountLineDiscountManaged) Clone() AmountLineDiscountManaged {
	return AmountLineDiscountManaged{
		ManagedModelWithID: i.ManagedModelWithID,
		AmountLineDiscount: i.AmountLineDiscount.Clone(),
	}
}

func (i AmountLineDiscountManaged) ContentsEqual(other AmountLineDiscountManaged) bool {
	return i.AmountLineDiscount.Equal(other.AmountLineDiscount)
}

func (i AmountLineDiscountManaged) GetManagedFieldsWithID() models.ManagedModelWithID {
	return i.ManagedModelWithID
}

func (i AmountLineDiscountManaged) WithManagedFieldsWithID(managed models.ManagedModelWithID) AmountLineDiscountManaged {
	return AmountLineDiscountManaged{
		ManagedModelWithID: managed,
		AmountLineDiscount: i.AmountLineDiscount.Clone(),
	}
}

type AmountLineDiscountsManaged []AmountLineDiscountManaged

func (i AmountLineDiscountsManaged) Clone() AmountLineDiscountsManaged {
	return lo.Map(i, func(item AmountLineDiscountManaged, _ int) AmountLineDiscountManaged {
		return item.Clone()
	})
}

func (i AmountLineDiscountsManaged) SumAmount(currency currencyx.Calculator) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, amount := range i {
		sum = sum.Add(currency.RoundToPrecision(amount.Amount)).Add(currency.RoundToPrecision(amount.RoundingAmount))
	}

	return sum
}

func (i AmountLineDiscountsManaged) GetByID() (map[string]AmountLineDiscountManaged, error) {
	out := make(map[string]AmountLineDiscountManaged, len(i))

	for _, amount := range i {
		out[amount.ID] = amount
	}

	return out, nil
}

func (i AmountLineDiscountsManaged) GetDiscountByChildUniqueReferenceID(childUniqueReferenceID string) (AmountLineDiscountManaged, bool) {
	for _, amount := range i {
		if amount.ChildUniqueReferenceID != nil && *amount.ChildUniqueReferenceID == childUniqueReferenceID {
			return amount, true
		}
	}

	return AmountLineDiscountManaged{}, false
}

func (i AmountLineDiscountsManaged) Mutate(mutator func(AmountLineDiscountManaged) (AmountLineDiscountManaged, error)) (AmountLineDiscountsManaged, error) {
	cloned := i.Clone()

	for idx := range cloned {
		mutated, err := mutator(cloned[idx])
		if err != nil {
			return nil, err
		}

		cloned[idx] = mutated
	}

	return cloned, nil
}

func (i AmountLineDiscountsManaged) Validate() error {
	var errs []error

	for _, amount := range i {
		if err := amount.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("amount[%s]: %w", amount.ID, err))
		}
	}

	return errors.Join(errs...)
}

func (i AmountLineDiscountsManaged) ReuseIDsFrom(existingItems AmountLineDiscountsManaged) AmountLineDiscountsManaged {
	return ReuseIDsFrom(i, existingItems)
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
	return deriveEqualUsageLineDiscount(&i, &other)
}

func (i UsageLineDiscount) Clone() UsageLineDiscount {
	return UsageLineDiscount{
		LineDiscountBase:      i.LineDiscountBase.Clone(),
		Quantity:              i.Quantity,
		PreLinePeriodQuantity: i.PreLinePeriodQuantity,
	}
}

type UsageLineDiscountManaged struct {
	models.ManagedModelWithID `json:",inline"`
	UsageLineDiscount         `json:",inline"`
}

func (i UsageLineDiscountManaged) Validate() error {
	var errs []error

	if err := i.UsageLineDiscount.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (i UsageLineDiscountManaged) Equal(other UsageLineDiscountManaged) bool {
	return deriveEqualUsageLineDiscountManaged(&i, &other)
}

func (i UsageLineDiscountManaged) Clone() UsageLineDiscountManaged {
	return UsageLineDiscountManaged{
		ManagedModelWithID: i.ManagedModelWithID,
		UsageLineDiscount:  i.UsageLineDiscount.Clone(),
	}
}

func (i UsageLineDiscountManaged) ContentsEqual(other UsageLineDiscountManaged) bool {
	return i.UsageLineDiscount.Equal(other.UsageLineDiscount)
}

func (i UsageLineDiscountManaged) GetManagedFieldsWithID() models.ManagedModelWithID {
	return i.ManagedModelWithID
}

func (i UsageLineDiscountManaged) WithManagedFieldsWithID(managed models.ManagedModelWithID) UsageLineDiscountManaged {
	return UsageLineDiscountManaged{
		ManagedModelWithID: managed,
		UsageLineDiscount:  i.UsageLineDiscount.Clone(),
	}
}

var _ models.Clonable[UsageLineDiscountsManaged] = (*UsageLineDiscountsManaged)(nil)

type UsageLineDiscountsManaged []UsageLineDiscountManaged

func (d UsageLineDiscountsManaged) Clone() UsageLineDiscountsManaged {
	return lo.Map(d, func(item UsageLineDiscountManaged, _ int) UsageLineDiscountManaged {
		return item.Clone()
	})
}

func (d UsageLineDiscountsManaged) MergeDiscountsByChildUniqueReferenceID(newDiscount UsageLineDiscountManaged) UsageLineDiscountsManaged {
	out := d.Clone()
	if newDiscount.ChildUniqueReferenceID == nil {
		return append(out, newDiscount)
	}

	oldDiscount, idx, ok := lo.FindIndexOf(out, func(item UsageLineDiscountManaged) bool {
		if item.ChildUniqueReferenceID == nil {
			return false
		}

		return *item.ChildUniqueReferenceID == *newDiscount.ChildUniqueReferenceID
	})
	if !ok {
		// No existing discount found with this child unique reference ID, let's add it
		return append(out, newDiscount)
	}

	out[idx] = newDiscount.WithManagedFieldsWithID(
		models.ManagedModelWithID{
			ID: oldDiscount.ID,
			ManagedModel: models.ManagedModel{
				CreatedAt: oldDiscount.CreatedAt,
				// UpdatedAt is updated by the adapter layer
				// DeletedAt should not be set, to ensure that we are not carrying over soft-deletion flags
			},
		},
	)

	return out
}

// LineDiscounts is a list of line discounts.

var (
	_ models.Clonable[LineDiscounts] = (*LineDiscounts)(nil)
	_ models.Validator               = (*LineDiscounts)(nil)
)

type LineDiscounts struct {
	Amount AmountLineDiscountsManaged `json:"amount,omitempty"`
	Usage  UsageLineDiscountsManaged  `json:"usage,omitempty"`
}

func (i LineDiscounts) Clone() LineDiscounts {
	out := LineDiscounts{}

	if len(i.Amount) > 0 {
		out.Amount = lo.Map(i.Amount, func(item AmountLineDiscountManaged, _ int) AmountLineDiscountManaged {
			return item.Clone()
		})
	}

	if len(i.Usage) > 0 {
		out.Usage = lo.Map(i.Usage, func(item UsageLineDiscountManaged, _ int) UsageLineDiscountManaged {
			return item.Clone()
		})
	}

	return out
}

func (i LineDiscounts) Validate() error {
	var errs []error

	for _, amount := range i.Amount {
		if err := amount.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("amount[%s]: %w", amount.ID, err))
		}
	}

	for _, usage := range i.Usage {
		if err := usage.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("usage[%s]: %w", usage.ID, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i LineDiscounts) ReuseIDsFrom(existingItems LineDiscounts) LineDiscounts {
	return LineDiscounts{
		Amount: ReuseIDsFrom(i.Amount, existingItems.Amount),
		Usage:  ReuseIDsFrom(i.Usage, existingItems.Usage),
	}
}

func (i LineDiscounts) IsEmpty() bool {
	return len(i.Amount) == 0 && len(i.Usage) == 0
}

type entityWithReusableIDs[T any] interface {
	GetChildUniqueReferenceID() *string
	GetManagedFieldsWithID() models.ManagedModelWithID
	WithManagedFieldsWithID(models.ManagedModelWithID) T
}

// ReuseIDsFrom reuses the IDs of the existing discounts by child unique reference ID.
func ReuseIDsFrom[T entityWithReusableIDs[T]](currentItems []T, dbExistingItems []T) []T {
	if len(currentItems) == 0 {
		return nil
	}

	existingItemsByUniqueReference := lo.GroupBy(
		lo.Filter(dbExistingItems, func(item T, _ int) bool {
			return item.GetChildUniqueReferenceID() != nil
		}),
		func(item T) string {
			return *item.GetChildUniqueReferenceID()
		},
	)

	discountsWithIDReuse := lo.Map(currentItems, func(discount T, _ int) T {
		childUniqueReferenceID := discount.GetChildUniqueReferenceID()

		// We should not reuse the ID if they are for a different child unique reference ID
		if childUniqueReferenceID == nil {
			return discount
		}

		existingItems, ok := existingItemsByUniqueReference[*childUniqueReferenceID]
		if !ok {
			// We did not find any existing items for this child unique reference ID,
			// let's create a new entry in the DB.
			return discount
		}

		existingManagedFields := existingItems[0].GetManagedFieldsWithID()

		return discount.WithManagedFieldsWithID(models.ManagedModelWithID{
			ID: existingManagedFields.ID,
			ManagedModel: models.ManagedModel{
				CreatedAt: existingManagedFields.CreatedAt,
				// UpdatedAt is updated by the adapter layer
				// DeletedAt should not be set, to ensure that we are not carrying over soft-deletion flags
			},
		})
	})

	return slicesx.EmptyAsNil(discountsWithIDReuse)
}

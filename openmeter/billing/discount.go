package billing

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type discountType[T any] interface {
	models.Clonable[T]
	models.Equaler[T]
	models.Validator
}

// Extended discount types

var _ discountType[PercentageDiscount] = (*PercentageDiscount)(nil)

type PercentageDiscount struct {
	productcatalog.PercentageDiscount `json:",inline"`

	CorrelationID string `json:"correlationID"`
}

func (d PercentageDiscount) Clone() PercentageDiscount {
	return PercentageDiscount{
		PercentageDiscount: d.PercentageDiscount.Clone(),
		CorrelationID:      d.CorrelationID,
	}
}

func (d PercentageDiscount) Equal(other PercentageDiscount) bool {
	if d.PercentageDiscount.Hash() != other.PercentageDiscount.Hash() {
		return false
	}

	if d.CorrelationID != other.CorrelationID {
		return false
	}

	return true
}

type UsageDiscount struct {
	productcatalog.UsageDiscount `json:",inline"`

	CorrelationID string `json:"correlationID"`
}

var _ discountType[UsageDiscount] = (*UsageDiscount)(nil)

func (d UsageDiscount) Clone() UsageDiscount {
	return UsageDiscount{
		UsageDiscount: d.UsageDiscount.Clone(),
		CorrelationID: d.CorrelationID,
	}
}

func (d UsageDiscount) Equal(other UsageDiscount) bool {
	if d.UsageDiscount.Hash() != other.UsageDiscount.Hash() {
		return false
	}

	if d.CorrelationID != other.CorrelationID {
		return false
	}

	return true
}

var _ models.Clonable[Discounts] = (*Discounts)(nil)

type Discounts struct {
	Percentage *PercentageDiscount `json:"percentage,omitempty"`
	Usage      *UsageDiscount      `json:"usage,omitempty"`
}

func (d Discounts) Clone() Discounts {
	discounts := Discounts{}

	if d.Percentage != nil {
		discounts.Percentage = lo.ToPtr(d.Percentage.Clone())
	}

	if d.Usage != nil {
		discounts.Usage = lo.ToPtr(d.Usage.Clone())
	}

	return discounts
}

func (d Discounts) IsEmpty() bool {
	return lo.IsEmpty(d)
}

func (d Discounts) ValidateForPrice(price productcatalog.Price) error {
	if d.Percentage != nil {
		return d.Percentage.ValidateForPrice(&price)
	}

	if d.Usage != nil {
		return d.Usage.ValidateForPrice(&price)
	}

	return nil
}

// DiscountReason type
type discountReason interface {
	json.Marshaler
	json.Unmarshaler

	models.Validator

	Type() DiscountReasonType
	AsRatecardPercentage() (PercentageDiscount, error)
	AsRatecardUsage() (UsageDiscount, error)
	AsMaximumSpend() (MaximumSpendDiscount, error)
}

var _ discountReason = (*DiscountReason)(nil)

type DiscountReasonType string

const (
	MaximumSpendDiscountReason       DiscountReasonType = "maximum_spend"
	RatecardPercentageDiscountReason DiscountReasonType = "ratecard_percentage"
	RatecardUsageDiscountReason      DiscountReasonType = "ratecard_usage"
)

func (DiscountReasonType) Values() []string {
	return []string{
		string(MaximumSpendDiscountReason),
		string(RatecardPercentageDiscountReason),
		string(RatecardUsageDiscountReason),
	}
}

// MaximumSpendDiscount contains information about the maximum spend induced discounts
type MaximumSpendDiscount struct{}

type DiscountReason struct {
	t DiscountReasonType

	percentage *PercentageDiscount
	usage      *UsageDiscount
}

func NewDiscountReasonFrom[T PercentageDiscount | UsageDiscount | productcatalog.PercentageDiscount | productcatalog.UsageDiscount | MaximumSpendDiscount](in T) DiscountReason {
	switch d := any(in).(type) {
	case PercentageDiscount:
		percentage := any(d).(PercentageDiscount)
		return DiscountReason{
			t:          RatecardPercentageDiscountReason,
			percentage: &percentage,
		}
	case productcatalog.PercentageDiscount:
		percentage := any(d).(productcatalog.PercentageDiscount)
		return DiscountReason{
			t: RatecardPercentageDiscountReason,
			percentage: &PercentageDiscount{
				PercentageDiscount: percentage,
			},
		}
	case UsageDiscount:
		usage := any(d).(UsageDiscount)
		return DiscountReason{
			t:     RatecardUsageDiscountReason,
			usage: &usage,
		}
	case productcatalog.UsageDiscount:
		usage := any(d).(productcatalog.UsageDiscount)
		return DiscountReason{
			t: RatecardUsageDiscountReason,
			usage: &UsageDiscount{
				UsageDiscount: usage,
			},
		}
	case MaximumSpendDiscount:
		return DiscountReason{
			t: MaximumSpendDiscountReason,
		}
	}

	return DiscountReason{}
}

func (d *DiscountReason) MarshalJSON() ([]byte, error) {
	var serde interface{}

	switch d.t {
	case RatecardPercentageDiscountReason:
		serde = struct {
			Type DiscountReasonType `json:"type"`
			*PercentageDiscount
		}{
			Type:               RatecardPercentageDiscountReason,
			PercentageDiscount: d.percentage,
		}
	case RatecardUsageDiscountReason:
		serde = struct {
			Type DiscountReasonType `json:"type"`
			*UsageDiscount
		}{
			Type:          RatecardUsageDiscountReason,
			UsageDiscount: d.usage,
		}
	case MaximumSpendDiscountReason:
		serde = struct {
			Type DiscountReasonType `json:"type"`
		}{
			Type: MaximumSpendDiscountReason,
		}
	default:
		return nil, fmt.Errorf("invalid Discount type: %s", d.t)
	}

	b, err := json.Marshal(serde)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON serialize Discount: %w", err)
	}

	return b, nil
}

func (d *DiscountReason) UnmarshalJSON(bytes []byte) error {
	serde := &struct {
		Type DiscountReasonType `json:"type"`
	}{}

	if err := json.Unmarshal(bytes, serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize Discount type: %w", err)
	}

	switch serde.Type {
	case RatecardPercentageDiscountReason:
		v := &PercentageDiscount{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize Discount: %w", err)
		}

		d.percentage = v
		d.t = RatecardPercentageDiscountReason
	case RatecardUsageDiscountReason:
		v := &UsageDiscount{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize Discount: %w", err)
		}

		d.usage = v
		d.t = RatecardUsageDiscountReason
	case MaximumSpendDiscountReason:
		d.t = MaximumSpendDiscountReason
	default:
		return fmt.Errorf("invalid Discount type: %s", serde.Type)
	}

	return nil
}

func (d *DiscountReason) Type() DiscountReasonType {
	return d.t
}

func (d *DiscountReason) AsRatecardPercentage() (PercentageDiscount, error) {
	if d.t != RatecardPercentageDiscountReason {
		return PercentageDiscount{}, errors.New("invalid discount type")
	}

	if d.percentage == nil {
		return PercentageDiscount{}, errors.New("percentage discount is missing")
	}

	return *d.percentage, nil
}

func (d *DiscountReason) AsRatecardUsage() (UsageDiscount, error) {
	if d.t != RatecardUsageDiscountReason {
		return UsageDiscount{}, errors.New("invalid discount type")
	}

	if d.usage == nil {
		return UsageDiscount{}, errors.New("usage discount is missing")
	}

	return *d.usage, nil
}

func (d *DiscountReason) AsMaximumSpend() (MaximumSpendDiscount, error) {
	if d.t != MaximumSpendDiscountReason {
		return MaximumSpendDiscount{}, errors.New("invalid discount type")
	}

	return MaximumSpendDiscount{}, nil
}

func (d *DiscountReason) Validate() error {
	switch d.t {
	case RatecardPercentageDiscountReason:
		return d.percentage.Validate()
	case RatecardUsageDiscountReason:
		return d.usage.Validate()
	case MaximumSpendDiscountReason:
		return nil
	default:
		return fmt.Errorf("invalid discount type: %s", d.t)
	}
}

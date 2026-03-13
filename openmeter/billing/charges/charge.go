package charges

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

type Charge struct {
	t meta.ChargeType

	flatFee        *flatfee.Charge
	creditPurchase *creditpurchase.Charge
}

func (c Charge) Type() meta.ChargeType {
	return c.t
}

func NewCharge[T flatfee.Charge | creditpurchase.Charge](ch T) Charge {
	switch v := any(ch).(type) {
	case flatfee.Charge:
		return Charge{
			t:       meta.ChargeTypeFlatFee,
			flatFee: &v,
		}
	case creditpurchase.Charge:
		return Charge{
			t:              meta.ChargeTypeCreditPurchase,
			creditPurchase: &v,
		}
	}

	return Charge{}
}

func (c Charge) Validate() error {
	switch c.t {
	case meta.ChargeTypeFlatFee:
		if c.flatFee == nil {
			return fmt.Errorf("flat fee charge is nil")
		}

		return c.flatFee.Validate()
	case meta.ChargeTypeCreditPurchase:
		if c.creditPurchase == nil {
			return fmt.Errorf("credit purchase charge is nil")
		}

		return c.creditPurchase.Validate()
	}

	return fmt.Errorf("invalid charge type: %s", c.t)
}

func (c Charge) AsFlatFeeCharge() (flatfee.Charge, error) {
	if c.t != meta.ChargeTypeFlatFee {
		return flatfee.Charge{}, fmt.Errorf("charge is not a flat fee charge")
	}

	return *c.flatFee, nil
}

func (c Charge) AsCreditPurchaseCharge() (creditpurchase.Charge, error) {
	if c.t != meta.ChargeTypeCreditPurchase {
		return creditpurchase.Charge{}, fmt.Errorf("charge is not a credit purchase charge")
	}

	return *c.creditPurchase, nil
}

func (c Charge) GetChargeID() (meta.ChargeID, error) {
	switch c.t {
	case meta.ChargeTypeFlatFee:
		if c.flatFee == nil {
			return meta.ChargeID{}, fmt.Errorf("flat fee charge is nil")
		}

		return c.flatFee.GetChargeID(), nil
	case meta.ChargeTypeCreditPurchase:
		if c.creditPurchase == nil {
			return meta.ChargeID{}, fmt.Errorf("credit purchase charge is nil")
		}

		return c.creditPurchase.GetChargeID(), nil
	}

	return meta.ChargeID{}, fmt.Errorf("invalid charge type: %s", c.t)
}

type Charges []Charge

func (c Charges) Validate() error {
	var errs []error

	for i, ch := range c {
		if err := ch.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("charge [%d]: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

type ChargeIntent struct {
	t meta.ChargeType

	flatFee        *flatfee.Intent
	creditPurchase *creditpurchase.Intent
}

func NewChargeIntent[T flatfee.Intent | creditpurchase.Intent](ch T) ChargeIntent {
	switch v := any(ch).(type) {
	case flatfee.Intent:
		return ChargeIntent{
			t:       meta.ChargeTypeFlatFee,
			flatFee: &v,
		}
	case creditpurchase.Intent:
		return ChargeIntent{
			t:              meta.ChargeTypeCreditPurchase,
			creditPurchase: &v,
		}
	}

	return ChargeIntent{}
}

func (i ChargeIntent) Type() meta.ChargeType {
	return i.t
}

func (i ChargeIntent) Validate() error {
	switch i.t {
	case meta.ChargeTypeFlatFee:
		if i.flatFee == nil {
			return fmt.Errorf("flat fee is nil")
		}

		return i.flatFee.Validate()
	case meta.ChargeTypeCreditPurchase:
		if i.creditPurchase == nil {
			return fmt.Errorf("credit purchase is nil")
		}

		return i.creditPurchase.Validate()
	}

	return fmt.Errorf("invalid charge type: %s", i.t)
}

func (i ChargeIntent) AsFlatFeeIntent() (flatfee.Intent, error) {
	if i.t != meta.ChargeTypeFlatFee {
		return flatfee.Intent{}, fmt.Errorf("charge is not a flat fee charge")
	}

	if i.flatFee == nil {
		return flatfee.Intent{}, fmt.Errorf("flat fee is nil")
	}

	return *i.flatFee, nil
}

func (i ChargeIntent) AsCreditPurchaseIntent() (creditpurchase.Intent, error) {
	if i.t != meta.ChargeTypeCreditPurchase {
		return creditpurchase.Intent{}, fmt.Errorf("charge is not a credit purchase charge")
	}

	if i.creditPurchase == nil {
		return creditpurchase.Intent{}, fmt.Errorf("credit purchase is nil")
	}

	return *i.creditPurchase, nil
}

type ChargeIntents []ChargeIntent

func (i ChargeIntents) Validate() error {
	var errs []error

	for idx, ch := range i {
		if err := ch.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}

type ChargeIntentsByType struct {
	FlatFee        []WithIndex[flatfee.Intent]
	CreditPurchase []WithIndex[creditpurchase.Intent]
}

func (i ChargeIntents) ByType() (ChargeIntentsByType, error) {
	out := ChargeIntentsByType{
		FlatFee:        make([]WithIndex[flatfee.Intent], 0, len(i)),
		CreditPurchase: make([]WithIndex[creditpurchase.Intent], 0, len(i)),
	}

	for idx, ch := range i {
		switch ch.Type() {
		case meta.ChargeTypeFlatFee:
			flatFee, err := ch.AsFlatFeeIntent()
			if err != nil {
				return ChargeIntentsByType{}, err
			}

			out.FlatFee = append(out.FlatFee, WithIndex[flatfee.Intent]{Index: idx, Value: flatFee})
		case meta.ChargeTypeCreditPurchase:
			creditPurchase, err := ch.AsCreditPurchaseIntent()
			if err != nil {
				return ChargeIntentsByType{}, err
			}

			out.CreditPurchase = append(out.CreditPurchase, WithIndex[creditpurchase.Intent]{Index: idx, Value: creditPurchase})
		default:
			return ChargeIntentsByType{}, fmt.Errorf("unsupported charge type: %s", ch.Type())
		}
	}

	return out, nil
}

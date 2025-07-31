package billingworkersubscription

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// This file contains temporary helpers to handle the flat fee -> ubp flat fee transition
// should be removed once we have fully migrated to the new flat fee line structure

func isFlatFee(line *billing.Line) bool {
	if line == nil {
		return false
	}

	if line.Type == billing.InvoiceLineTypeUsageBased &&
		line.UsageBased != nil &&
		line.UsageBased.Price != nil &&
		line.UsageBased.Price.Type() == productcatalog.FlatPriceType {
		return true
	}

	return false
}

func getFlatFeePerUnitAmount(line *billing.Line) (alpacadecimal.Decimal, error) {
	if line == nil {
		return alpacadecimal.Zero, fmt.Errorf("line is nil")
	}

	if line.Type != billing.InvoiceLineTypeUsageBased {
		return alpacadecimal.Zero, fmt.Errorf("line is not a usage based line")
	}

	if line.UsageBased == nil || line.UsageBased.Price == nil {
		return alpacadecimal.Zero, fmt.Errorf("line misses usage based metadata")
	}

	flatPrice, err := line.UsageBased.Price.AsFlat()
	if err != nil {
		return alpacadecimal.Zero, err
	}

	return flatPrice.Amount, nil
}

func setFlatFeePerUnitAmount(line *billing.Line, perUnitAmount alpacadecimal.Decimal) error {
	if line == nil {
		return fmt.Errorf("line is nil")
	}

	if line.Type != billing.InvoiceLineTypeUsageBased {
		return fmt.Errorf("line is not a usage based line")
	}

	if line.UsageBased == nil || line.UsageBased.Price == nil {
		return fmt.Errorf("line misses usage based metadata")
	}

	flatPrice, err := line.UsageBased.Price.AsFlat()
	if err != nil {
		return err
	}

	flatPrice.Amount = perUnitAmount
	line.UsageBased.Price = productcatalog.NewPriceFrom(flatPrice)
	return nil
}

type typeWithEqual[T any] interface {
	Equal(T) bool
}

func setIfDoesNotEqual[T typeWithEqual[T]](existing *T, expected T, wasChange *bool) {
	if !(*existing).Equal(expected) {
		*existing = expected
		*wasChange = true
	}
}

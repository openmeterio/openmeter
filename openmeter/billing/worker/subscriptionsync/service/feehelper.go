package service

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func isFlatFee(line billing.GenericInvoiceLineReader) bool {
	if line == nil {
		return false
	}

	price := line.GetPrice()
	if price == nil {
		return false
	}

	if price.Type() == productcatalog.FlatPriceType {
		return true
	}

	return false
}

func getFlatFeePerUnitAmount(line billing.GenericInvoiceLineReader) (alpacadecimal.Decimal, error) {
	if line == nil {
		return alpacadecimal.Zero, fmt.Errorf("line is nil")
	}

	price := line.GetPrice()

	if price == nil {
		return alpacadecimal.Zero, fmt.Errorf("line misses usage based metadata")
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return alpacadecimal.Zero, err
	}

	return flatPrice.Amount, nil
}

func setFlatFeePerUnitAmount(line billing.GenericInvoiceLine, perUnitAmount alpacadecimal.Decimal) error {
	if line == nil {
		return fmt.Errorf("line is nil")
	}

	price := line.GetPrice()
	if price == nil {
		return fmt.Errorf("line misses usage based metadata")
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return err
	}

	flatPrice.Amount = perUnitAmount
	line.SetPrice(lo.FromPtr(productcatalog.NewPriceFrom(flatPrice)))
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

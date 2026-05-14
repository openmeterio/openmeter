package invoiceupdater

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func IsFlatFee(line billing.GenericInvoiceLineReader) bool {
	if line == nil {
		return false
	}

	price := line.GetPrice()
	if price == nil {
		return false
	}

	return price.Type() == productcatalog.FlatPriceType
}

func GetFlatFeePerUnitAmount(line billing.GenericInvoiceLineReader) (alpacadecimal.Decimal, error) {
	if line == nil {
		return alpacadecimal.Zero, fmt.Errorf("line is nil")
	}

	price := line.GetPrice()
	if price == nil {
		return alpacadecimal.Zero, fmt.Errorf("line missing flat-fee metadata")
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return alpacadecimal.Zero, err
	}

	return flatPrice.Amount, nil
}

func SetFlatFeePerUnitAmount(line billing.GenericInvoiceLine, perUnitAmount alpacadecimal.Decimal) error {
	if line == nil {
		return fmt.Errorf("line is nil")
	}

	price := line.GetPrice()
	if price == nil {
		return fmt.Errorf("line missing flat-fee metadata")
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return err
	}

	flatPrice.Amount = perUnitAmount
	line.SetPrice(lo.FromPtr(productcatalog.NewPriceFrom(flatPrice)))
	return nil
}

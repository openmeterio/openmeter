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

	if line.Type == billing.InvoiceLineTypeFee {
		return true
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

	switch line.Type {
	case billing.InvoiceLineTypeFee:
		if line.FlatFee == nil {
			return alpacadecimal.Zero, fmt.Errorf("line misses flat fee metadata")
		}

		return line.FlatFee.PerUnitAmount, nil
	case billing.InvoiceLineTypeUsageBased:
		if line.UsageBased == nil || line.UsageBased.Price == nil {
			return alpacadecimal.Zero, fmt.Errorf("line misses usage based metadata")
		}

		flatPrice, err := line.UsageBased.Price.AsFlat()
		if err != nil {
			return alpacadecimal.Zero, err
		}

		return flatPrice.Amount, nil
	default:
		return alpacadecimal.Zero, fmt.Errorf("line is not a (flat or usage based) fee line")
	}
}

func setFlatFeePerUnitAmount(line *billing.Line, perUnitAmount alpacadecimal.Decimal) error {
	if line == nil {
		return fmt.Errorf("line is nil")
	}

	switch line.Type {
	case billing.InvoiceLineTypeFee:
		if line.FlatFee == nil {
			return fmt.Errorf("line misses flat fee metadata")
		}

		line.FlatFee.PerUnitAmount = perUnitAmount
		return nil
	case billing.InvoiceLineTypeUsageBased:
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
	default:
		return fmt.Errorf("line is not a (flat or usage based) fee line")
	}
}

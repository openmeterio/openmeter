package testutils

import (
	"fmt"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func NewChargesEnabledLineRouter(t testing.TB) billing.CreateLineRouter {
	t.Helper()

	return chargesEnabledLineRouter{}
}

type chargesEnabledLineRouter struct{}

func (chargesEnabledLineRouter) GetLineEngineForCreateLine(line billing.GenericInvoiceLineReader) (billing.LineEngineType, error) {
	if line == nil {
		return "", fmt.Errorf("line is required")
	}

	price := line.GetPrice()
	if price == nil {
		return "", fmt.Errorf("line[%s]: price is required", line.GetID())
	}

	switch price.Type() {
	case productcatalog.FlatPriceType:
		return billing.LineEngineTypeChargeFlatFee, nil
	default:
		return billing.LineEngineTypeChargeUsageBased, nil
	}
}

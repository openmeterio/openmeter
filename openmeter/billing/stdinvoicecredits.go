package billing

import (
	"errors"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreditsApplied []CreditApplied

var (
	_ models.Validator                = (*CreditsApplied)(nil)
	_ models.Clonable[CreditsApplied] = (*CreditsApplied)(nil)
)

func (c CreditsApplied) Validate() error {
	for _, item := range c {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c CreditsApplied) Clone() CreditsApplied {
	return lo.Map(c, func(item CreditApplied, _ int) CreditApplied {
		return item
	})
}

func (c CreditsApplied) SumAmount(currency currencyx.Calculator) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, item := range c {
		sum = sum.Add(currency.RoundToPrecision(item.Amount))
	}

	return sum
}

type CreditApplied struct {
	Amount              alpacadecimal.Decimal `json:"amount"`
	Description         string                `json:"description"`
	CreditRealizationID string                `json:"creditRealizationID"`

	// TODO[later]: Once we see the overall structure we might want to add references
}

func (c CreditApplied) CloneWithAmount(amount alpacadecimal.Decimal) CreditApplied {
	c.Amount = amount
	return c
}

func (c CreditApplied) Validate() error {
	if !c.Amount.IsPositive() {
		return errors.New("amount must be positive")
	}

	return nil
}

package ledger

import "fmt"

// ----------------------------------------------------------------------------
// Typed SubAccounts
// ----------------------------------------------------------------------------

// CustomerFBOSubAccount is a sub-account that is a customer FBO sub-account.
type CustomerFBOSubAccount struct {
	SubAccount
}

func AsCustomerFBOSubAccount(s SubAccount) (*CustomerFBOSubAccount, error) {
	if s.Address().AccountType() != AccountTypeCustomerFBO {
		return nil, fmt.Errorf("sub-account is not a customer FBO sub-account")
	}
	return &CustomerFBOSubAccount{
		SubAccount: s,
	}, nil
}

func (c *CustomerFBOSubAccount) CustomerDimensions() (CustomerSubAccountDimensions, error) {
	dim := c.SubAccount.Dimensions()

	res := CustomerSubAccountDimensions{
		Currency: dim.Currency,
		Features: dim.Feature,
	}

	var ok bool

	res.TaxCode, ok = dim.TaxCode.Get()
	if !ok {
		return CustomerSubAccountDimensions{}, fmt.Errorf("tax code is required")
	}

	res.CreditPriority, ok = dim.CreditPriority.Get()
	if !ok {
		return CustomerSubAccountDimensions{}, fmt.Errorf("credit priority is required")
	}

	return res, nil
}

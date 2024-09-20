package openmetersandbox

type TaxConfiguration struct{}

func (t *TaxConfiguration) Validate() error {
	return nil
}

type InvoicingConfiguration struct{}

func (t *InvoicingConfiguration) Validate() error {
	return nil
}

type PaymentConfiguration struct{}

func (t *PaymentConfiguration) Validate() error {
	return nil
}

type TaxState struct{}

func (t *TaxState) Validate() error {
	return nil
}

type InvoicingState struct{}

func (t *InvoicingState) Validate() error {
	return nil
}

type PaymentState struct{}

func (t *PaymentState) Validate() error {
	return nil
}

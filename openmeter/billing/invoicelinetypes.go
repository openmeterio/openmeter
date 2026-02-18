package billing

// TODO: Deprecated: remove in the next PR
type InvoiceLineAdapterType string

const (
	// InvoiceLineTypeFee is an item that represents a single charge without meter backing.
	InvoiceLineAdapterTypeFee InvoiceLineAdapterType = "flat_fee"
	// InvoiceLineTypeUsageBased is an item that is added to the invoice and is usage based.
	InvoiceLineAdapterTypeUsageBased InvoiceLineAdapterType = "usage_based"
)

func (InvoiceLineAdapterType) Values() []string {
	return []string{
		string(InvoiceLineAdapterTypeFee),
		string(InvoiceLineAdapterTypeUsageBased),
	}
}

// TODO: Deprecated: remove in the next PR
type InvoiceLineStatus string

const (
	// InvoiceLineStatusValid is a valid invoice line.
	InvoiceLineStatusValid InvoiceLineStatus = "valid"
	// InvoiceLineStatusDetailed is a detailed invoice line.
	InvoiceLineStatusDetailed InvoiceLineStatus = "detailed"
)

func (InvoiceLineStatus) Values() []string {
	return []string{
		string(InvoiceLineStatusValid),
		string(InvoiceLineStatusDetailed),
	}
}

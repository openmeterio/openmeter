package billingentity

type StripeTaxCode string

type StripeTaxOverride struct {
	TaxCode StripeTaxCode `json:"taxCode,omitempty"`
}

type TaxOverrides struct {
	Stripe *StripeTaxOverride `json:"stripe,omitempty"`
}

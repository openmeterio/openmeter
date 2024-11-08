package billingentity

type StripeTaxCode string

type StripeTaxOverride struct {
	TaxCode StripeTaxCode `json:"taxCode,omitempty"`
}

// TODO[OM-979]: This (and the mappers) should come from product catalog when the API/entities are available
type TaxOverrides struct {
	Stripe *StripeTaxOverride `json:"stripe,omitempty"`
}

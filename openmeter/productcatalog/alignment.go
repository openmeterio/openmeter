package productcatalog

type Alignment struct {
	// BillablesMustAlign indicates whether all billable items in a given phase must share the same BillingPeriodDuration.
	BillablesMustAlign bool `json:"billablesMustAlign"`
}

// AlignmentUpdate is used for the nil-ish comparison of the plan service only
type AlignmentUpdate struct {
	BillablesMustAlign *bool `json:"billablesMustAlign,omitempty"`
}

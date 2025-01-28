package productcatalog

type Alignment struct {
	BillablesMustAlign bool `json:"billablesMustAlign"`
}

// AlignmentUpdate is used for the nil-ish comparison of the plan service only
type AlignmentUpdate struct {
	BillablesMustAlign *bool `json:"billablesMustAlign,omitempty"`
}

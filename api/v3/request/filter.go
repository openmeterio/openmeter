package request

// Query string format to be supported:
// ?filter[field][operator][quantifier]

type FilterOperator struct {
	eq        *string   `query:"eq"`
	neq       *string   `query:"neq"`
	gt        *string   `query:"gt"`
	gte       *string   `query:"gte"`
	lt        *string   `query:"lt"`
	lte       *string   `query:"lte"`
	contains  *string   `query:"contains"`
	ocontains *[]string `query:"ocontains"`
	exists    *bool     `query:"exists"`
	oeq       *[]string `query:"oeq"`
}

//type FilterQuantifier struct {
//	any *string `query:"any"`
//	all *string `query:"all"`
//}
//
//type QueryFilter struct {
//	Field  string
//	Filter FilterOperator
//}

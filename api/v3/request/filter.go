package request

// Query string format to be supported:
// ?filter[field][operator][quantifier]

type FilterOperator struct {
	Eq        *string   `query:"eq"`
	Neq       *string   `query:"neq"`
	Gt        *string   `query:"gt"`
	Gte       *string   `query:"gte"`
	Lt        *string   `query:"lt"`
	Lte       *string   `query:"lte"`
	Contains  *string   `query:"contains"`
	Ocontains *[]string `query:"ocontains"`
	Exists    *bool     `query:"exists"`
	Oeq       *[]string `query:"oeq"`
}

//type FilterQuantifier struct {
//	any *string `query:"any"`
//	all *string `query:"all"`
//}

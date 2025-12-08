package request

type Filter struct {
	eq        *string   `query:"eq"`
	neq       *string   `query:"neq"`
	gt        *string   `query:"gt"`
	gte       *string   `query:"gte"`
	lt        *string   `query:"lt"`
	lte       *string   `query:"lte"`
	contains  *[]string `query:"contains"`
	ocontains *[]string `query:"ocontains"`
	exists    *bool     `query:"exists"`
	oeq       *string   `query:"oeq"`
}

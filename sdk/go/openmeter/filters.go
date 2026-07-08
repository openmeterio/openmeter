package openmeter

// StringFilter expresses a comparison on a string field. Set exactly one
// operator; unset fields are omitted from the request. It mirrors the API's
// FilterString type. (The spec also allows a bare string shorthand for equality
// on query-string filters, e.g. filter[key]=value; that is a server-side parse
// convenience, not a distinct JSON shape — use Eq for the same effect.)
type StringFilter struct {
	Eq        *string  `json:"eq,omitempty"`
	Neq       *string  `json:"neq,omitempty"`
	Gt        *string  `json:"gt,omitempty"`
	Gte       *string  `json:"gte,omitempty"`
	Lt        *string  `json:"lt,omitempty"`
	Lte       *string  `json:"lte,omitempty"`
	Contains  *string  `json:"contains,omitempty"`
	Oeq       []string `json:"oeq,omitempty"`
	Ocontains []string `json:"ocontains,omitempty"`
	Exists    *bool    `json:"$exists,omitempty"`
}

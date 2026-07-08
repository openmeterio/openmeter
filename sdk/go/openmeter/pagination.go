package openmeter

// PageParams selects a page of a paginated listing.
type PageParams struct {
	// Size is the number of items per page.
	Size *int
	// Number is the 1-based page number.
	Number *int
}

// PageMeta carries the pagination query parameters echoed back plus the total
// count. The API types these as JSON numbers; the SDK exposes them as ints.
type PageMeta struct {
	Number int `json:"number"`
	Size   int `json:"size"`
	Total  int `json:"total"`
}

// PaginatedMeta wraps the pagination information of a page-paginated response.
type PaginatedMeta struct {
	Page PageMeta `json:"page"`
}

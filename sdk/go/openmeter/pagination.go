package openmeter

import (
	"iter"
	"net/url"
	"strconv"
)

// defaultListPageSize is the page size the auto-paginating iterators request
// when the caller does not specify one, chosen to keep round-trips low without
// oversized responses.
const defaultListPageSize = 100

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

// addPageParams serializes page pagination as deepObject query members
// (page[size], page[number]). It is shared by every list endpoint that uses the
// page-pagination style.
func addPageParams(q url.Values, page *PageParams) {
	if page == nil {
		return
	}

	if page.Size != nil {
		setDeepObjectString(q, "page", "size", strconv.Itoa(*page.Size))
	}
	if page.Number != nil {
		setDeepObjectString(q, "page", "number", strconv.Itoa(*page.Number))
	}
}

// paginate drives page-by-page iteration over a list endpoint and exposes it as
// a Go 1.23 range-over-func iterator. start seeds the first page number and page
// size (defaulting to page 1 and defaultListPageSize); fetch retrieves one page
// and reports its items plus the server's reported total.
//
// On a fetch error the iterator yields one (zero-value, err) pair and stops.
// Iteration also stops on an empty page or once the reported total has been
// seen; the total guard is skipped when the server reports a non-positive total
// so a bad count cannot end paging early. Breaking out of the range loop stops
// paging.
func paginate[T any](start *PageParams, fetch func(page, size int) (data []T, total int, err error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		page, size := 1, defaultListPageSize
		if start != nil {
			if start.Number != nil {
				page = *start.Number
			}
			if start.Size != nil {
				size = *start.Size
			}
		}

		seen := 0
		for {
			data, total, err := fetch(page, size)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}

			for _, item := range data {
				if !yield(item, nil) {
					return
				}
			}

			seen += len(data)

			if len(data) == 0 {
				return
			}
			if total > 0 && seen >= total {
				return
			}

			page++
		}
	}
}

package request

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

const (
	PageNumberQuery = "page[number]"

	// offset pagination specific
	PageSizeQuery = "page[size]"

	// cursor pagination specific
	PageBeforeQuery = "page[before]"
	PageAfterQuery  = "page[after]"

	DefaultPaginationNumber = 1
	DefaultPaginationSize   = 20
)

var (
	ErrCursorUndefined = errors.New("at least before or after cursor need to be defined")
	ErrCursorRange     = errors.New("range pagination not supported, both before and after cursor were defined")
)

type Pagination struct {
	Size   int
	Number int
	Offset int
	Limit  int
	After  *Cursor
	Before *Cursor
}

func extractPagination(ctx context.Context, qs url.Values, c *config) (Pagination, *apierrors.BaseAPIError) {
	size, apiErr := parsePageSize(ctx, qs, c)
	if apiErr != nil {
		return Pagination{}, apiErr
	}

	p := Pagination{Size: min(size, c.maxPageSize)}

	switch c.paginationKind {
	case paginationKindOffset:
		number, apiErr := parsePageNumber(ctx, qs, c)
		if apiErr != nil {
			return Pagination{}, apiErr
		}
		p.Number = number
		coef := max(p.Number-1, 0)
		p.Offset = coef * p.Size
		p.Limit = p.Size

	case paginationKindCursor:
		if qs.Has(PageBeforeQuery) && qs.Has(PageAfterQuery) {
			return Pagination{}, apierrors.NewBadRequestError(ctx, ErrCursorRange,
				apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Source: apierrors.InvalidParamSourceQuery,
						Reason: "api doesn't support range pagination",
					},
				})
		}

		before, apiErr := parseCursorParam(ctx, qs, PageBeforeQuery, c.cursorCipherKey, c.cursorValidateUUIDs)
		if apiErr != nil {
			return Pagination{}, apiErr
		}
		p.Before = before

		after, apiErr := parseCursorParam(ctx, qs, PageAfterQuery, c.cursorCipherKey, c.cursorValidateUUIDs)
		if apiErr != nil {
			return Pagination{}, apiErr
		}
		p.After = after
	}

	return p, nil
}

func parsePageSize(ctx context.Context, qs url.Values, c *config) (int, *apierrors.BaseAPIError) {
	if !qs.Has(PageSizeQuery) {
		return c.defaultPageSize, nil
	}

	pageSize, err := strconv.ParseInt(qs.Get(PageSizeQuery), 10, 16)
	if err != nil && (c.strictMode || pageSize < 0) {
		return 0, apierrors.NewBadRequestError(ctx, err,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Field:  PageSizeQuery,
					Reason: "unable to parse query field",
					Source: apierrors.InvalidParamSourceQuery,
					Rule:   "page size should be a positive integer",
				},
			})
	}
	if err != nil {
		return c.defaultPageSize, nil
	}
	if pageSize < 1 {
		return DefaultPaginationSize, nil
	}
	return int(pageSize), nil
}

func parsePageNumber(ctx context.Context, qs url.Values, c *config) (int, *apierrors.BaseAPIError) {
	if !qs.Has(PageNumberQuery) {
		return DefaultPaginationNumber, nil
	}

	pageNumber, err := strconv.ParseInt(qs.Get(PageNumberQuery), 10, 16)
	if err != nil && c.strictMode {
		return 0, apierrors.NewBadRequestError(ctx, err,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Field:  PageNumberQuery,
					Reason: "unable to parse query field",
					Source: apierrors.InvalidParamSourceQuery,
					Rule:   "page number should be a positive integer",
				},
			})
	}
	if err == nil && pageNumber < 0 {
		return 0, apierrors.NewBadRequestError(ctx, err,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Field:  PageNumberQuery,
					Reason: "unable to parse query field",
					Source: apierrors.InvalidParamSourceQuery,
					Rule:   "page number should be a positive integer",
				},
			})
	}
	if err != nil || pageNumber < 1 {
		return DefaultPaginationNumber, nil
	}
	return int(pageNumber), nil
}

func parseCursorParam(ctx context.Context, qs url.Values, key, cipherKey string, validateUUIDs bool) (*Cursor, *apierrors.BaseAPIError) {
	if !qs.Has(key) {
		return nil, nil
	}
	cursor, err := decodeCursorAfterQueryUnescape(cipherKey, qs.Get(key), validateUUIDs)
	if err != nil {
		return nil, apierrors.NewBadRequestError(ctx, err,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Source: apierrors.InvalidParamSourceQuery,
					Reason: fmt.Sprintf("unable to parse %s cursor", key),
				},
			})
	}
	return cursor, nil
}

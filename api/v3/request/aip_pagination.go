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
	p := Pagination{
		Size: c.defaultPageSize,
	}

	if qs.Has(PageSizeQuery) {
		strPageSize := qs.Get(PageSizeQuery)
		pageSize, err := strconv.ParseInt(strPageSize, 10, 16)
		if err != nil {
			if c.strictMode || pageSize < 0 {
				return p, apierrors.NewBadRequestError(ctx, err,
					apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  PageSizeQuery,
							Reason: "unable to parse query field",
							Source: apierrors.InvalidParamSourceQuery,
							Rule:   "page size should be a positive integer",
						},
					})
			} else {
				pageSize = int64(c.defaultPageSize)
			}
		}
		if pageSize < 1 {
			pageSize = DefaultPaginationSize
		}
		p.Size = int(pageSize)
	}

	if p.Size > c.maxPageSize {
		p.Size = c.maxPageSize
	}

	if c.paginationKind == paginationKindOffset {
		p.Number = DefaultPaginationNumber

		if qs.Has(PageNumberQuery) {
			strPageNumber := qs.Get(PageNumberQuery)
			pageNumber, err := strconv.ParseInt(strPageNumber, 10, 16)
			if err != nil {
				if c.strictMode || pageNumber < 0 {
					return p, apierrors.NewBadRequestError(ctx, err,
						apierrors.InvalidParameters{
							apierrors.InvalidParameter{
								Field:  PageNumberQuery,
								Reason: "unable to parse query field",
								Source: apierrors.InvalidParamSourceQuery,
								Rule:   "page number should be a positive integer",
							},
						})
				}
			}
			if pageNumber < 1 {
				pageNumber = DefaultPaginationNumber
			}
			p.Number = int(pageNumber)
		}

		var coef int
		coef = int(p.Number) - 1
		if coef < 0 {
			coef = 0
		}
		p.Offset = coef * p.Size
		p.Limit = p.Size
	} else if c.paginationKind == paginationKindCursor {
		if qs.Has(PageBeforeQuery) && qs.Has(PageAfterQuery) {
			return p, apierrors.NewBadRequestError(ctx, ErrCursorRange,
				apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Source: apierrors.InvalidParamSourceQuery,
						Reason: "api doesn't support range pagination",
					},
				})
		}

		if qs.Has(PageBeforeQuery) {
			b, err := decodeCursorAfterQueryUnescape(c.cursorCipherKey, qs.Get(PageBeforeQuery), c.cursorValidateUUIDs)
			if err != nil {
				return p, apierrors.NewBadRequestError(ctx, err,
					apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Source: apierrors.InvalidParamSourceQuery,
							Reason: fmt.Sprintf("unable to parse %s cursor", PageBeforeQuery),
						},
					})
			}
			p.Before = b
		}
		if qs.Has(PageAfterQuery) {
			a, err := decodeCursorAfterQueryUnescape(c.cursorCipherKey, qs.Get(PageAfterQuery), c.cursorValidateUUIDs)
			if err != nil {
				return p, apierrors.NewBadRequestError(ctx, err,
					apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Source: apierrors.InvalidParamSourceQuery,
							Reason: fmt.Sprintf("unable to parse %s cursor", PageAfterQuery),
						},
					})
			}
			p.After = a
		}
	}

	return p, nil
}

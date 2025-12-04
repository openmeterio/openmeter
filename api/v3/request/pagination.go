package request

import (
	"errors"

	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const (
	DefaultPaginationSize = 20
	PageBeforeQuery       = "page[before]"
	PageAfterQuery        = "page[after]"
)

var (
	ErrCursorPaginationSizeInvalid = errors.New("size must be greater than 0")
	ErrCursorPaginationUndefined   = errors.New("at least before or after cursor need to be defined")
	ErrCursorPaginationRange       = errors.New("range pagination not supported, both before and after cursor were defined")
)

type CursorPagination struct {
	Size   int
	After  *pagination.Cursor
	Before *pagination.Cursor
}

func (p *CursorPagination) Validate() error {
	if p.Size < 1 {
		return ErrCursorPaginationSizeInvalid
	}

	if p.After == nil && p.Before == nil {
		return ErrCursorPaginationUndefined
	}

	if p.After != nil && p.Before != nil {
		return ErrCursorPaginationRange
	}

	return nil
}

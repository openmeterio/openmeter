package request

import (
	"errors"

	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type paginationKind string

const (
	paginationKindPage   paginationKind = "page"
	paginationKindCursor paginationKind = "cursor"
)

const (
	DefaultPaginationSize    = 20
	DefaultPaginationMaxSize = 100
	DefaultPaginationKind    = paginationKindPage
)

var (
	ErrCursorPaginationSizeInvalid   = errors.New("size must be greater than 0")
	ErrCursorPaginationRange         = errors.New("range pagination not supported, both before and after cursor were defined")
	ErrCursorPaginationAfterInvalid  = errors.New("after cursor is invalid")
	ErrCursorPaginationBeforeInvalid = errors.New("before cursor is invalid")
)

type Pagination struct {
	kind paginationKind

	// Cursor pagination
	Size   int                `query:"size"`
	After  *pagination.Cursor `query:"after"`
	Before *pagination.Cursor `query:"before"`

	// Offset pagination
	Number int `query:"number"`
}

func (p *Pagination) Validate() error {
	if p.Size < 1 {
		return ErrCursorPaginationSizeInvalid
	}

	if p.kind == paginationKindCursor {
		if p.After != nil && p.Before != nil {
			return ErrCursorPaginationRange
		}

		if p.After != nil && p.After.Validate() != nil {
			return ErrCursorPaginationAfterInvalid
		}

		if p.Before != nil && p.Before.Validate() != nil {
			return ErrCursorPaginationBeforeInvalid
		}
	}

	return nil
}

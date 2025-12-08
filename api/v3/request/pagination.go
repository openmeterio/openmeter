package request

import (
	"errors"

	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const (
	DefaultPaginationSize = 20
)

var (
	ErrCursorPaginationSizeInvalid = errors.New("size must be greater than 0")
	ErrCursorPaginationUndefined   = errors.New("at least before or after cursor need to be defined")
	ErrCursorPaginationRange       = errors.New("range pagination not supported, both before and after cursor were defined")
)

type CursorPagination struct {
	Size   int                `query:"size"`
	After  *pagination.Cursor `query:"after"`
	Before *pagination.Cursor `query:"before"`
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

package request

import (
	"errors"
	"strings"

	"github.com/openmeterio/openmeter/pkg/sortx"
)

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

var (
	ErrSortByInvalid     = errors.New("invalid sort by")
	ErrSortFieldRequired = errors.New("sort field is required")
	ErrSortOrderInvalid  = errors.New("sort order must be either asc or desc")
	defaultOrder         = SortOrderAsc
)

func (s SortOrder) Validate() error {
	if s != SortOrderAsc && s != SortOrderDesc {
		return ErrSortOrderInvalid
	}

	return nil
}

type SortBy struct {
	Field string
	Order SortOrder
}

func ParseSortBy(sort string) (*SortBy, error) {
	var s SortBy
	err := s.UnmarshalText([]byte(sort))
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (s SortBy) Validate() error {
	if s.Field == "" {
		return ErrSortFieldRequired
	}

	if err := s.Order.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *SortBy) UnmarshalText(text []byte) error {
	parts := strings.Fields(string(text))
	if len(parts) == 0 {
		return ErrSortByInvalid
	}
	if len(parts) > 2 {
		return ErrSortByInvalid
	}

	s.Field = parts[0]
	if len(parts) == 2 {
		s.Order = SortOrder(parts[1])
	} else {
		s.Order = defaultOrder
	}

	return s.Validate()
}

func (s SortOrder) ToSortxOrder() sortx.Order {
	if s == SortOrderAsc {
		return sortx.OrderAsc
	}

	return sortx.OrderDesc
}

package filters

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
)

// ErrInvalidDateTime is a sentinel when a value can't be parsed as RFC-3339.
var ErrInvalidDateTime = errors.New("invalid datetime")

func invalidDateTimeError(op, value string) error {
	return fmt.Errorf("%s filter: %w: %q", op, ErrInvalidDateTime, value)
}

// FromAPIFilterString converts an API FilterString to filter.FilterString.
func FromAPIFilterString(f *FilterString) (*filter.FilterString, error) {
	if f == nil {
		return nil, nil
	}

	var parts []filter.FilterString

	if f.Eq != nil {
		parts = append(parts, filter.FilterString{Eq: f.Eq})
	}
	if f.Neq != nil {
		parts = append(parts, filter.FilterString{Ne: f.Neq})
	}
	if f.Exists != nil {
		parts = append(parts, filter.FilterString{Exists: f.Exists})
	}
	if f.Contains != nil {
		parts = append(parts, filter.FilterString{Contains: f.Contains})
	}
	if f.Gt != nil {
		parts = append(parts, filter.FilterString{Gt: f.Gt})
	}
	if f.Gte != nil {
		parts = append(parts, filter.FilterString{Gte: f.Gte})
	}
	if f.Lt != nil {
		parts = append(parts, filter.FilterString{Lt: f.Lt})
	}
	if f.Lte != nil {
		parts = append(parts, filter.FilterString{Lte: f.Lte})
	}
	if len(f.Oeq) > 0 {
		parts = append(parts, filter.FilterString{In: convert.SliceToPointer(f.Oeq)})
	}
	if len(f.Ocontains) > 0 {
		parts = append(parts, filter.FilterString{
			Or: convert.SliceToPointer(lo.Map(f.Ocontains, func(v string, _ int) filter.FilterString {
				return filter.FilterString{Contains: &v}
			})),
		})
	}

	switch len(parts) {
	case 0:
		return nil, nil
	case 1:
		return &parts[0], nil
	default:
		return &filter.FilterString{And: &parts}, nil
	}
}

// FromAPIFilterStringExact converts an API FilterStringExact to filter.FilterString.
func FromAPIFilterStringExact(f *FilterStringExact) (*filter.FilterString, error) {
	if f == nil {
		return nil, nil
	}

	return &filter.FilterString{
		Eq: f.Eq,
		Ne: f.Neq,
		In: &f.Oeq,
	}, nil
}

// FromAPIFilterNumeric converts an API FilterNumeric to filter.FilterFloat.
func FromAPIFilterNumeric(f *FilterNumeric) (*filter.FilterFloat, error) {
	if f == nil {
		return nil, nil
	}

	var parts []filter.FilterFloat

	if f.Eq != nil {
		parts = append(parts, filter.FilterFloat{Eq: f.Eq})
	}
	if f.Neq != nil {
		parts = append(parts, filter.FilterFloat{Ne: f.Neq})
	}
	if f.Gt != nil {
		parts = append(parts, filter.FilterFloat{Gt: f.Gt})
	}
	if f.Gte != nil {
		parts = append(parts, filter.FilterFloat{Gte: f.Gte})
	}
	if f.Lt != nil {
		parts = append(parts, filter.FilterFloat{Lt: f.Lt})
	}
	if f.Lte != nil {
		parts = append(parts, filter.FilterFloat{Lte: f.Lte})
	}
	if len(f.Oeq) > 0 {
		parts = append(parts, filter.FilterFloat{
			Or: convert.SliceToPointer(lo.Map(f.Oeq, func(v float64, _ int) filter.FilterFloat {
				return filter.FilterFloat{Eq: &v}
			})),
		})
	}

	switch len(parts) {
	case 0:
		return nil, nil
	case 1:
		return &parts[0], nil
	default:
		return &filter.FilterFloat{And: &parts}, nil
	}
}

// FromAPIFilterDateTime converts an API FilterDateTime to filter.FilterTime.
func FromAPIFilterDateTime(f *FilterDateTime) (*filter.FilterTime, error) {
	if f == nil {
		return nil, nil
	}

	var parts []filter.FilterTime

	if f.Eq != nil {
		t, err := time.Parse(time.RFC3339, *f.Eq)
		if err != nil {
			return nil, invalidDateTimeError("eq", *f.Eq)
		}
		parts = append(parts, filter.FilterTime{Eq: &t})
	}
	if f.Gt != nil {
		t, err := time.Parse(time.RFC3339, *f.Gt)
		if err != nil {
			return nil, invalidDateTimeError("gt", *f.Gt)
		}
		parts = append(parts, filter.FilterTime{Gt: &t})
	}
	if f.Gte != nil {
		t, err := time.Parse(time.RFC3339, *f.Gte)
		if err != nil {
			return nil, invalidDateTimeError("gte", *f.Gte)
		}
		parts = append(parts, filter.FilterTime{Gte: &t})
	}
	if f.Lt != nil {
		t, err := time.Parse(time.RFC3339, *f.Lt)
		if err != nil {
			return nil, invalidDateTimeError("lt", *f.Lt)
		}
		parts = append(parts, filter.FilterTime{Lt: &t})
	}
	if f.Lte != nil {
		t, err := time.Parse(time.RFC3339, *f.Lte)
		if err != nil {
			return nil, invalidDateTimeError("lte", *f.Lte)
		}
		parts = append(parts, filter.FilterTime{Lte: &t})
	}

	switch len(parts) {
	case 0:
		return nil, nil
	case 1:
		return &parts[0], nil
	default:
		return &filter.FilterTime{And: &parts}, nil
	}
}

// FromAPIFilterBoolean converts an API FilterBoolean to filter.FilterBoolean.
func FromAPIFilterBoolean(f *FilterBoolean) (*filter.FilterBoolean, error) {
	if f == nil {
		return nil, nil
	}

	return &filter.FilterBoolean{
		Eq: f.Eq,
	}, nil
}

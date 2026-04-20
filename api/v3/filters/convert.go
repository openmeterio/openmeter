package filters

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
)

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

// FromAPIFilterULID converts an API FilterString to filter.FilterULID.
func FromAPIFilterULID(f *FilterULID) (*filter.FilterULID, error) {
	if f == nil {
		return nil, nil
	}

	var parts []filter.FilterULID

	if f.Eq != nil {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{Eq: f.Eq}})
	}
	if f.Neq != nil {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{Ne: f.Neq}})
	}
	if f.Exists != nil {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{Exists: f.Exists}})
	}
	if f.Contains != nil {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{Contains: f.Contains}})
	}
	if f.Gt != nil {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{Gt: f.Gt}})
	}
	if f.Gte != nil {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{Gte: f.Gte}})
	}
	if f.Lt != nil {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{Lt: f.Lt}})
	}
	if f.Lte != nil {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{Lte: f.Lte}})
	}
	if len(f.Oeq) > 0 {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{In: convert.SliceToPointer(f.Oeq)}})
	}
	if len(f.Ocontains) > 0 {
		parts = append(parts, filter.FilterULID{FilterString: filter.FilterString{
			Or: convert.SliceToPointer(lo.Map(f.Ocontains, func(v string, _ int) filter.FilterString {
				return filter.FilterString{Contains: &v}
			})),
		}})
	}

	switch len(parts) {
	case 0:
		return nil, nil
	case 1:
		return &parts[0], nil
	default:
		return &filter.FilterULID{And: &parts}, nil
	}
}

// FromAPIFilterLabel converts an API FilterLabel to filter.FilterString.
func FromAPIFilterLabel(f *FilterLabel) (*filter.FilterString, error) {
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
	if f.Contains != nil {
		parts = append(parts, filter.FilterString{Contains: f.Contains})
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

// FromAPIFilterLabels converts an API FilterLabels to map[string]filter.FilterString.
func FromAPIFilterLabels(f *FilterLabels) (map[string]filter.FilterString, error) {
	if f == nil {
		return nil, nil
	}

	parts := make(map[string]filter.FilterString, len(*f))
	for k, v := range *f {
		ff, err := FromAPIFilterLabel(&v)
		if err != nil {
			return nil, err
		}
		if ff == nil {
			continue
		}
		parts[k] = *ff
	}

	return parts, nil
}

// FromAPIFilterStringExact converts an API FilterStringExact to filter.FilterString.
func FromAPIFilterStringExact(f *FilterStringExact) (*filter.FilterString, error) {
	if f == nil {
		return nil, nil
	}

	out := &filter.FilterString{
		Eq: f.Eq,
		Ne: f.Neq,
	}
	if len(f.Oeq) > 0 {
		out.In = &f.Oeq
	}
	return out, nil
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
		parts = append(parts, filter.FilterTime{Eq: f.Eq})
	}
	if f.Gt != nil {
		parts = append(parts, filter.FilterTime{Gt: f.Gt})
	}
	if f.Gte != nil {
		parts = append(parts, filter.FilterTime{Gte: f.Gte})
	}
	if f.Lt != nil {
		parts = append(parts, filter.FilterTime{Lt: f.Lt})
	}
	if f.Lte != nil {
		parts = append(parts, filter.FilterTime{Lte: f.Lte})
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

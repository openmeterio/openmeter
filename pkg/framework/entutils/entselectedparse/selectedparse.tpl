{{/*
    This template generates helpers for parsing appended selected values
    (for example from JOIN aliases) into typed ent nodes using generated
    scanValues/assignValues code.
*/}}
{{ define "selectedparse" }}
{{ $pkg := base $.Config.Package }}
{{ template "header" $ }}

func assignSelectedValue(dst any, src ent.Value) error {
	switch d := dst.(type) {
	case *sql.NullString:
		switch v := src.(type) {
		case nil:
			*d = sql.NullString{}
			return nil
		case sql.NullString:
			*d = v
			return nil
		case *sql.NullString:
			if v == nil {
				*d = sql.NullString{}
				return nil
			}
			*d = *v
			return nil
		case string:
			*d = sql.NullString{String: v, Valid: true}
			return nil
		case []byte:
			*d = sql.NullString{String: string(v), Valid: true}
			return nil
		default:
			return fmt.Errorf("cannot assign %T to %T", src, dst)
		}
	case *sql.NullTime:
		switch v := src.(type) {
		case nil:
			*d = sql.NullTime{}
			return nil
		case sql.NullTime:
			*d = v
			return nil
		case *sql.NullTime:
			if v == nil {
				*d = sql.NullTime{}
				return nil
			}
			*d = *v
			return nil
		case time.Time:
			*d = sql.NullTime{Time: v, Valid: true}
			return nil
		case *time.Time:
			if v == nil {
				*d = sql.NullTime{}
				return nil
			}
			*d = sql.NullTime{Time: *v, Valid: true}
			return nil
		default:
			return fmt.Errorf("cannot assign %T to %T", src, dst)
		}
	case *[]byte:
		switch v := src.(type) {
		case nil:
			*d = nil
			return nil
		case []byte:
			*d = append((*d)[:0], v...)
			return nil
		case string:
			*d = []byte(v)
			return nil
		default:
			return fmt.Errorf("cannot assign %T to %T", src, dst)
		}
	default:
		if src == nil {
			return nil
		}
		dv := reflect.ValueOf(dst)
		if dv.Kind() != reflect.Pointer || dv.IsNil() {
			return fmt.Errorf("destination must be a non-nil pointer: %T", dst)
		}
		sv := reflect.ValueOf(src)
		ev := dv.Elem()
		if sv.Type().AssignableTo(ev.Type()) {
			ev.Set(sv)
			return nil
		}
		if sv.Type().ConvertibleTo(ev.Type()) {
			ev.Set(sv.Convert(ev.Type()))
			return nil
		}
		return fmt.Errorf("cannot assign %T to %T", src, dst)
	}
}

func isNullSelectedValue(v ent.Value) bool {
	switch val := v.(type) {
	case nil:
		return true
	case sql.NullString:
		return !val.Valid
	case *sql.NullString:
		return val == nil || !val.Valid
	case sql.NullTime:
		return !val.Valid
	case *sql.NullTime:
		return val == nil || !val.Valid
	case sql.NullInt64:
		return !val.Valid
	case *sql.NullInt64:
		return val == nil || !val.Valid
	case sql.NullFloat64:
		return !val.Valid
	case *sql.NullFloat64:
		return val == nil || !val.Valid
	case sql.NullBool:
		return !val.Valid
	case *sql.NullBool:
		return val == nil || !val.Valid
	default:
		return false
	}
}

{{ range $n := $.Nodes }}
func Parse{{ $n.Name }}FromSelectedValues(prefix string, getValue func(string) (ent.Value, error)) (*{{ $n.Name }}, error) {
	idValue, err := getValue(prefix + {{ $n.Package }}.FieldID)
	if err != nil {
		return nil, fmt.Errorf("read selected value %q: %w", prefix+{{ $n.Package }}.FieldID, err)
	}
	if isNullSelectedValue(idValue) {
		return nil, nil
	}

	columns := {{ $n.Package }}.Columns
	scanValues := (*{{ $n.Name }}).scanValues
	values, err := scanValues(nil, columns)
	if err != nil {
		return nil, fmt.Errorf("prepare scan values: %w", err)
	}

	for i := range columns {
		selectedName := prefix + columns[i]
		value, err := getValue(selectedName)
		if err != nil {
			return nil, fmt.Errorf("read selected value %q: %w", selectedName, err)
		}
		if err := assignSelectedValue(values[i], value); err != nil {
			return nil, fmt.Errorf("assign selected value %q: %w", selectedName, err)
		}
	}

	node := &{{ $n.Name }}{}
	assignValues := (*{{ $n.Name }}).assignValues
	if err := assignValues(node, columns, values); err != nil {
		return nil, fmt.Errorf("assign selected values to {{ $n.Name }}: %w", err)
	}

	return node, nil
}
{{ end }}

{{ end }}

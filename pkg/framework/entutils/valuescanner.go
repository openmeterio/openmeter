package entutils

import (
	"database/sql"
	"database/sql/driver"

	"entgo.io/ent/schema/field"
	"github.com/samber/lo"
)

// NillableValueScannerFunc wraps a ValueScannerFunc and allows for scanning NULL values.
func NillableValueScannerFunc[T any](valueScanner field.ValueScannerFunc[T, *sql.NullString]) field.ValueScannerFunc[*T, *sql.NullString] {
	return field.ValueScannerFunc[*T, *sql.NullString]{
		V: func(value *T) (driver.Value, error) {
			if value == nil {
				return sql.NullString{}, nil
			}
			v, err := valueScanner.V(*value)
			if err != nil {
				return nil, err
			}
			return &v, nil
		},
		S: func(value *sql.NullString) (*T, error) {
			if !value.Valid {
				return nil, nil
			}

			v, err := valueScanner.S(value)
			if err != nil {
				return nil, err
			}

			return lo.ToPtr(v), nil
		},
	}
}

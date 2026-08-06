package entutils

import (
	"database/sql/driver"
	"encoding/json"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"github.com/samber/mo"
)

func JSONStringValueScanner[T any]() field.ValueScannerFunc[T, *sql.NullString] {
	return field.ValueScannerFunc[T, *sql.NullString]{
		V: func(t T) (driver.Value, error) {
			return json.Marshal(t)
		},
		S: func(ns *sql.NullString) (T, error) {
			v := new(T)
			if ns == nil || !ns.Valid {
				return *v, nil
			}

			b := []byte(ns.String)
			if err := json.Unmarshal(b, v); err != nil {
				return *v, err
			}

			return *v, nil
		},
	}
}

type jsonStringArrayOption[T any] interface {
	Get() (T, bool)
}

// JSONStringArrayOptionValueScanner maps SQL NULL to an absent option and JSON
// arrays to a present option. Present nil arrays and JSON null are normalized
// to an empty array so they remain distinguishable from SQL NULL on later writes.
// The named option wrapper keeps generic option types compatible with Ent codegen.
func JSONStringArrayOptionValueScanner[T ~[]E, E any, O jsonStringArrayOption[T]](
	fromOption func(mo.Option[T]) O,
) field.ValueScannerFunc[O, *sql.NullString] {
	return field.ValueScannerFunc[O, *sql.NullString]{
		V: func(option O) (driver.Value, error) {
			value, ok := option.Get()
			if !ok {
				return nil, nil
			}

			if value == nil {
				value = make(T, 0)
			}

			return json.Marshal(value)
		},
		S: func(ns *sql.NullString) (O, error) {
			if ns == nil || !ns.Valid {
				return fromOption(mo.None[T]()), nil
			}

			value := make(T, 0)
			if err := json.Unmarshal([]byte(ns.String), &value); err != nil {
				return fromOption(mo.None[T]()), err
			}

			if value == nil {
				value = make(T, 0)
			}

			return fromOption(mo.Some(value)), nil
		},
	}
}

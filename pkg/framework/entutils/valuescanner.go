package entutils

import (
	"database/sql/driver"
	"encoding/json"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
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

package entedge

import "github.com/openmeterio/openmeter/openmeter/ent/db"

func OrNilIfNotFound[T any](edgeValue *T, err error) (*T, error) {
	if err != nil {
		if db.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return edgeValue, nil
}

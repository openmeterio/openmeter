package adapter

import (
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/models"
)

// MapChargeConstraintError translates an ent DB constraint violation errors.
func MapChargeConstraintError(err error) error {
	if err == nil || !entdb.IsConstraintError(err) {
		return err
	}

	switch {
	case sqlgraph.IsUniqueConstraintError(err):
		return models.NewGenericConflictError(
			fmt.Errorf("charge conflicts with an existing charge: %w", err),
		)
	case sqlgraph.IsForeignKeyConstraintError(err):
		return models.NewGenericValidationError(
			fmt.Errorf("charge references a resource that does not exist: %w", err),
		)
	default:
		return models.NewGenericValidationError(
			fmt.Errorf("charge violates a database constraint: %w", err),
		)
	}
}

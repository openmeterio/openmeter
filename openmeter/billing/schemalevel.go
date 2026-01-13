package billing

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SchemaLevels struct {
	ReadSchemaLevel  int
	WriteSchemaLevel int
}

var _ models.Validator = (*SchemaLevels)(nil)

func (s SchemaLevels) Validate() error {
	var errs []error
	if s.ReadSchemaLevel <= 0 {
		errs = append(errs, fmt.Errorf("read schema level is required"))
	}

	if s.WriteSchemaLevel <= 0 {
		errs = append(errs, fmt.Errorf("write schema level is required"))
	}

	if s.ReadSchemaLevel > s.WriteSchemaLevel {
		errs = append(errs, fmt.Errorf("read schema level is greater than write schema level: downgrades are not supported"))
	}

	return errors.Join(errs...)
}

package flatfee

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/pkg/models"
)

type DetailedLine = stddetailedline.Base

type DetailedLines []DetailedLine

func (l DetailedLines) Clone() DetailedLines {
	return lo.Map(l, func(dl DetailedLine, _ int) DetailedLine {
		return dl.Clone()
	})
}

func (l DetailedLines) Validate() error {
	var errs []error

	for idx, line := range l {
		if err := line.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

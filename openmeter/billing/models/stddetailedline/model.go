package stddetailedline

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Category string

const (
	// CategoryRegular is a regular flat fee, that is based on the usage or a subscription.
	CategoryRegular Category = "regular"
	// CategoryCommitment is a flat fee that is based on a commitment such as min spend.
	CategoryCommitment Category = "commitment"
)

func (Category) Values() []string {
	return []string{
		string(CategoryRegular),
		string(CategoryCommitment),
	}
}

var _ models.Validator = (*Category)(nil)

func (c Category) Validate() error {
	if !slices.Contains(Category("").Values(), string(c)) {
		return fmt.Errorf("invalid category %s", c)
	}

	return nil
}

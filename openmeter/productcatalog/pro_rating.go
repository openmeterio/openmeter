package productcatalog

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	ProRatingModeProratePrices ProRatingMode = "prorate_prices"
)

type ProRatingMode string

func (m ProRatingMode) Values() []string {
	return []string{
		string(ProRatingModeProratePrices),
	}
}

// ProRatingConfig defines the pro-rating behavior configuration.
type ProRatingConfig struct {
	// Enabled indicates whether pro-rating is enabled.
	Enabled bool `json:"enabled"`

	// Mode specifies how pro-rating should be calculated.
	Mode ProRatingMode `json:"mode"`
}

// Validate validates the ProRatingConfig.
func (p ProRatingConfig) Validate() error {
	var errs []error

	if !p.Enabled {
		return nil
	}

	switch p.Mode {
	case ProRatingModeProratePrices:
		// Valid mode
	default:
		errs = append(errs, fmt.Errorf("invalid Mode: %s", p.Mode))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Equal returns true if the two ProRatingConfigs are equal.
func (p ProRatingConfig) Equal(o ProRatingConfig) bool {
	return p.Enabled == o.Enabled && p.Mode == o.Mode
}

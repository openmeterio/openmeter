package config

import (
	"errors"
	"log/slog"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/datetime"
)

type EntitlementsConfiguration struct {
	// GracePeriod represents a period of time in the current usage period during which values will not be snapshotted.
	// This in effect results in late events being included during this period.
	GracePeriod datetime.ISODurationString
}

// Validate validates the configuration.
func (c EntitlementsConfiguration) Validate() error {
	if c.GracePeriod == "" {
		return errors.New("gracePeriod is required")
	}

	if p, err := c.GracePeriod.Parse(); err != nil {
		return errors.New("gracePeriod is invalid")
	} else if p.Sign() != 1 {
		return errors.New("gracePeriod must be positive")
	}

	return nil
}

func (c *EntitlementsConfiguration) GetGracePeriod() datetime.ISODuration {
	gracePeriod, err := c.GracePeriod.Parse()
	if err != nil {
		slog.Error("failed to parse grace period, using default of 1 day", "error", err)
		return datetime.NewISODuration(0, 0, 0, 1, 0, 0, 0)
	}
	return gracePeriod
}

func ConfigureEntitlements(v *viper.Viper, flags *pflag.FlagSet) {
	// Let's set the default grace period to 1 day so late events can arrive up to a day late and be included in balance calculations
	v.SetDefault("entitlements.gracePeriod", "P1D")
}

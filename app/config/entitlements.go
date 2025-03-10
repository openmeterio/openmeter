package config

import (
	"errors"
	"log/slog"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/isodate"
)

type EntitlementsConfiguration struct {
	GracePeriod isodate.String
}

// Validate validates the configuration.
func (c EntitlementsConfiguration) Validate() error {
	if c.GracePeriod == "" {
		return errors.New("gracePeriod is required")
	}

	if _, err := c.GracePeriod.Parse(); err != nil {
		return errors.New("gracePeriod is invalid")
	}

	return nil
}

func (c *EntitlementsConfiguration) GetGracePeriod() isodate.Period {
	gracePeriod, err := c.GracePeriod.Parse()
	if err != nil {
		slog.Error("failed to parse grace period, using default of 1 day", "error", err)
		return isodate.NewPeriod(0, 0, 0, 1, 0, 0, 0)
	}
	return gracePeriod
}

func ConfigureEntitlements(v *viper.Viper, flags *pflag.FlagSet) {
	// Let's set the default grace period to 1 day so late events can arrive up to a day late and be included in balance calculations
	v.SetDefault("entitlements.gracePeriod", "P1D")
}

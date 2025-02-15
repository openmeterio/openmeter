package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

type TerminationConfig struct {
	// CheckInterval defines the time period used for updating the readiness check based on the termination status
	CheckInterval time.Duration

	// GracefulShutdownTimeout defines the maximum time for the process to gracefully stop on receiving stop signal.
	GracefulShutdownTimeout time.Duration

	// PropagationTimeout defines how long to block the termination process in order
	// to allow the termination event to be propagated to other systems. e.g. reverse proxy.
	// Its value should be set higher than the failure threshold for readiness probe.
	// In Kubernetes it should be: readiness.periodSeconds * (readiness.failureThreshold + 1) + CheckInterval
	// PropagationTimeout must always less than GracefulShutdownTimeout.
	PropagationTimeout time.Duration
}

func (c TerminationConfig) Validate() error {
	var errs []error

	if c.CheckInterval < time.Second {
		errs = append(errs, errors.New("checkInterval must be greater than or equal to 1s"))
	}

	if c.PropagationTimeout > c.GracefulShutdownTimeout {
		errs = append(errs, errors.New("propagationTimeout must be less or equal to gracefulShutdownTimeout"))
	}

	return errors.Join(errs...)
}

// ConfigureTermination configures some defaults in the Viper instance.
func ConfigureTermination(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("checkInterval"), time.Second)
	v.SetDefault(prefixer("gracefulShutdownTimeout"), 30*time.Second)
	v.SetDefault(prefixer("propagationTimeout"), 3*time.Second)
}

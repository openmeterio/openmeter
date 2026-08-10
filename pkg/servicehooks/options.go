package servicehooks

import "fmt"

// CyclePolicy controls what happens when an invocation reaches a registration
// which is already active in the same context call chain.
type CyclePolicy int

const (
	// CyclePolicyError returns a CycleError before any hook in the nested
	// invocation runs. This is the default because cycles should be explicit.
	CyclePolicyError CyclePolicy = iota

	// CyclePolicySkip skips only the active registration. Other hooks in the
	// nested invocation still run in registration order.
	CyclePolicySkip
)

func (p CyclePolicy) validate() error {
	switch p {
	case CyclePolicyError, CyclePolicySkip:
		return nil
	default:
		return fmt.Errorf("%w: %d", ErrCyclePolicyInvalid, p)
	}
}

type registerConfig struct {
	cyclePolicy CyclePolicy
}

func defaultRegisterConfig() registerConfig {
	return registerConfig{
		cyclePolicy: CyclePolicyError,
	}
}

// RegisterOption configures one hook registration.
type RegisterOption func(*registerConfig) error

// WithCyclePolicy sets the behavior for a recursive invocation of this
// registration. CyclePolicyError is used when this option is absent.
func WithCyclePolicy(policy CyclePolicy) RegisterOption {
	return func(config *registerConfig) error {
		if err := policy.validate(); err != nil {
			return err
		}

		config.cyclePolicy = policy

		return nil
	}
}

package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/featuregate"
)

// alwaysFalseGate is a Gate that always returns false for any flag evaluation.
type alwaysFalseGate struct{}

func (alwaysFalseGate) EvaluateBool(_, _ string, _ bool) (bool, error) {
	return false, nil
}

func TestNewFeatureGateChecker_DisabledUsesNoop(t *testing.T) {
	checker := common.NewFeatureGateChecker(alwaysFalseGate{}, config.FeatureGateConfiguration{
		Enabled: false,
		Flags:   featuregate.Flags{featuregate.FeatureFlag("om_ff_credits_enabled"): "credits-flag"},
	})
	require.NotNil(t, checker)

	// With Enabled=false the checker wraps a Noop gate, so any flag evaluates to true
	got, err := checker.Enabled("ns", "credits-flag")
	require.NoError(t, err)
	assert.True(t, got, "disabled feature gate should use noop and return true")
}

func TestNewFeatureGateChecker_EnabledUsesRealGate(t *testing.T) {
	checker := common.NewFeatureGateChecker(alwaysFalseGate{}, config.FeatureGateConfiguration{
		Enabled: true,
		Flags:   featuregate.Flags{featuregate.FeatureFlag("om_ff_credits_enabled"): "credits-flag"},
	})
	require.NotNil(t, checker)

	// With Enabled=true the real gate is used, which always returns false
	got, err := checker.Enabled("ns", "credits-flag")
	require.NoError(t, err)
	assert.False(t, got, "enabled feature gate should use the real gate")
}

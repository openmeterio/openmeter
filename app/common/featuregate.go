package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/featuregate"
)

var FeatureGateChecker = wire.NewSet(
	featuregate.NewNoop,
	wire.FieldsOf(new(config.Configuration), "FeatureGate"),
	NewFeatureGateChecker,
)

func NewFeatureGateChecker(gate featuregate.Gate, config config.FeatureGateConfiguration) *featuregate.FeatureGateChecker {
	flags := make(featuregate.Flags)
	if config.Flags != nil {
		flags = config.Flags
	}

	if !config.Enabled {
		return featuregate.NewFeatureGateChecker(featuregate.NewNoop(), flags)
	}

	return featuregate.NewFeatureGateChecker(gate, flags)
}

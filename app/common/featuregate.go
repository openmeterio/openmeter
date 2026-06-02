package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/pkg/featuregate"
)

var FeatureGateNoopSet = wire.NewSet(
	featuregate.NewNoop,
)

package router

// We explicitly define no-op implementations for future APIs instead of just using the codegen version.

import (
	"github.com/openmeterio/openmeter/api"
)

var unimplemented api.ServerInterface = api.Unimplemented{}

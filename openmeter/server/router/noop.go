package router

// We explicitly define no-op implementations for future APIs instead of just using the codegen version.

import (
	apiv1 "github.com/openmeterio/openmeter/api"
)

var unimplemented apiv1.ServerInterface = apiv1.Unimplemented{}

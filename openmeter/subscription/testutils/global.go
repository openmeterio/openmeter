package subscriptiontestutils

import "github.com/openmeterio/openmeter/pkg/datex"

var ExampleNamespace = "test-namespace"

var ISOMonth, _ = datex.ISOString("P1M").Parse()

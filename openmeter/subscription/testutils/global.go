package subscriptiontestutils

import "github.com/openmeterio/openmeter/pkg/datetime"

var ExampleNamespace = "test-namespace"

var ISOMonth, _ = datetime.ISODurationString("P1M").Parse()

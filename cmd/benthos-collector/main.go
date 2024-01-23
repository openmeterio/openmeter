package main

import (
	"context"

	_ "github.com/benthosdev/benthos/v4/public/components/all"  // import all benthos components
	_ "github.com/benthosdev/benthos/v4/public/components/io"   // import io benthos components
	_ "github.com/benthosdev/benthos/v4/public/components/pure" // import pure benthos components
	"github.com/benthosdev/benthos/v4/public/service"

	_ "github.com/openmeterio/openmeter/collector/benthos/input"  // import input plugins
	_ "github.com/openmeterio/openmeter/collector/benthos/output" // import output plugins
)

func main() {
	service.RunCLI(context.Background())
}

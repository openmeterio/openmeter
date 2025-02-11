package main

import (
	"context"

	_ "github.com/redpanda-data/benthos/v4/public/components/io"
	_ "github.com/redpanda-data/benthos/v4/public/components/pure"
	_ "github.com/redpanda-data/benthos/v4/public/components/pure/extended"
	"github.com/redpanda-data/benthos/v4/public/service"
	_ "github.com/redpanda-data/connect/public/bundle/free/v4"

	_ "github.com/openmeterio/openmeter/collector/benthos/input"  // import input plugins
	_ "github.com/openmeterio/openmeter/collector/benthos/output" // import output plugins
	"github.com/openmeterio/openmeter/pkg/log"
)

func main() {
	defer log.PanicLogger(log.WithExit)

	service.RunCLI(context.Background())
}

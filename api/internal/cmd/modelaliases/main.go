// Command modelaliases generates the public model aliases for generated API transports.
package main

import (
	"fmt"
	"os"
)

const (
	v1ModelsImport = "github.com/openmeterio/openmeter/api/models"
	v3ModelsImport = "github.com/openmeterio/openmeter/api/v3/models"
)

var (
	v1Server = aliasTarget{packageName: "api", modelsImport: v1ModelsImport}
	v1Client = aliasTarget{packageName: "openmeter", modelsImport: v1ModelsImport, keepAuthKeys: true}
	v3Server = aliasTarget{packageName: "v3", modelsImport: v3ModelsImport}
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: modelaliases <v1-server|v1-client|v3-server>")
		os.Exit(2)
	}

	if err := generate(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "modelaliases:", err)
		os.Exit(1)
	}
}

func generate(target string) error {
	switch target {
	case "v1-server":
		return generateServerAliases(v1Server)
	case "v1-client":
		return generateV1ClientAliases()
	case "v3-server":
		return generateServerAliases(v3Server)
	default:
		return fmt.Errorf("unknown target %q", target)
	}
}

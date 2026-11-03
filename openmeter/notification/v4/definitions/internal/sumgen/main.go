// Command sumgen rewrites definitions.sum from the current definitions.
package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/openmeterio/openmeter/openmeter/notification/v4/definitions"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: sumgen <path to definitions.sum>")
		os.Exit(2)
	}

	// Generate into memory first, so a failure leaves the file untouched.
	var out bytes.Buffer
	if err := definitions.Generate(&out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := os.WriteFile(os.Args[1], out.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package main

// Provisioned by ldflags.
var version string

//nolint:gochecknoinits
func init() {
	if version == "" {
		version = "unknown"
	}
}

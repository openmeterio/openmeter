package common

import (
	"fmt"
	"runtime/debug"

	"go.opentelemetry.io/otel/attribute"

	"github.com/openmeterio/openmeter/app/config"
)

// Stamped by image builds when the source checkout has no VCS metadata.
var commitSHA string

// Metadata provides information about the service to components that need it (eg. telemetry).
type Metadata struct {
	ServiceName       string
	Version           string
	CommitSHA         string
	Environment       string
	OpenTelemetryName string

	AdditionalAttributes []attribute.KeyValue
}

func NewMetadata(conf config.Configuration, version string, serviceName string, additionalAttributes ...attribute.KeyValue) Metadata {
	return Metadata{
		ServiceName:          fmt.Sprintf("openmeter-%s", serviceName),
		Version:              version,
		CommitSHA:            writerCommitSHA(),
		Environment:          conf.Environment,
		OpenTelemetryName:    fmt.Sprintf("openmeter.io/%s", serviceName),
		AdditionalAttributes: additionalAttributes,
	}
}

func writerCommitSHA() string {
	if commitSHA != "" {
		return commitSHA
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}

	return ""
}

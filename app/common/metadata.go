package common

import (
	"fmt"

	"github.com/openmeterio/openmeter/app/config"
)

// Metadata provides information about the service to components that need it (eg. telemetry).
type Metadata struct {
	ServiceName       string
	Version           string
	Environment       string
	OpenTelemetryName string
}

func NewMetadata(conf config.Configuration, version string, serviceName string) Metadata {
	return Metadata{
		ServiceName:       fmt.Sprintf("openmeter-%s", serviceName),
		Version:           version,
		Environment:       conf.Environment,
		OpenTelemetryName: fmt.Sprintf("openmeter.io/%s", serviceName),
	}
}

package common

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/openmeterio/openmeter/app/config"
)

// Metadata provides information about the service to components that need it (eg. telemetry).
type Metadata struct {
	ServiceName       string
	Version           string
	Environment       string
	OpenTelemetryName string

	AdditionalAttributes []attribute.KeyValue
}

func NewMetadata(conf config.Configuration, version string, serviceName string, additionalAttributes ...attribute.KeyValue) Metadata {
	return Metadata{
		ServiceName:          fmt.Sprintf("openmeter-%s", serviceName),
		Version:              version,
		Environment:          conf.Environment,
		OpenTelemetryName:    fmt.Sprintf("openmeter.io/%s", serviceName),
		AdditionalAttributes: additionalAttributes,
	}
}

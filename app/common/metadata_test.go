package common

import (
	"testing"

	"github.com/stretchr/testify/require"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/openmeterio/openmeter/app/config"
)

func TestNewMetadata(t *testing.T) {
	conf := config.Configuration{Metadata: config.MetadataConfiguration{
		Version: "configured-version", Revision: "configured-revision",
	}}
	metadata := NewMetadata(conf, "binary-version", "test")
	require.Equal(t, "configured-revision", metadata.Revision)
	require.Equal(t, "configured-version", metadata.Version)

	serviceVersion, ok := NewTelemetryResource(metadata).Set().Value(semconv.ServiceVersionKey)
	require.True(t, ok)
	require.Equal(t, metadata.Version, serviceVersion.AsString())

	// Callers that supply a binary version without loading metadata config keep it.
	require.Equal(t, "binary-version", NewMetadata(config.Configuration{}, "binary-version", "test").Version)
}

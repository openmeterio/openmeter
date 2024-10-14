package common

// Metadata provides information about the service to components that need it (eg. telemetry).
type Metadata struct {
	ServiceName       string
	Version           string
	Environment       string
	OpenTelemetryName string
}

package common

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/oklog/run"
	"github.com/openmeterio/openmeter/app/config"
)

// Metadata provides information about the service to components that need it (eg. telemetry).
type Metadata struct {
	ServiceName       string
	Version           string
	Environment       string
	OpenTelemetryName string

	// Optional
	K8SPodUID string
}

func NewMetadata(conf config.Configuration, version string, otelName string) Metadata {
	return Metadata{
		ServiceName:       "openmeter",
		Version:           version,
		Environment:       conf.Environment,
		OpenTelemetryName: fmt.Sprintf("openmeter.io/%s", otelName),
		K8SPodUID:         conf.K8SPodUID,
	}
}

// Runner is a helper struct that runs a group of services.
type Runner struct {
	Group  run.Group
	Logger *slog.Logger
}

func (r Runner) Run() {
	err := r.Group.Run()
	if e := (run.SignalError{}); errors.As(err, &e) {
		r.Logger.Info("received signal: shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		r.Logger.Error("application stopped due to error", slog.String("error", err.Error()))
	}
}

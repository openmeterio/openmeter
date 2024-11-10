package common

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/oklog/run"
)

// Metadata provides information about the service to components that need it (eg. telemetry).
type Metadata struct {
	ServiceName       string
	Version           string
	Environment       string
	OpenTelemetryName string

	K8SPodUID *string
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

package common

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/oklog/run"
)

// Runner is a helper struct that runs a group of services.
type Runner struct {
	Group  run.Group
	Logger *slog.Logger
}

func (r Runner) Run() {
	if err := r.RunWithError(); err != nil {
		r.Logger.Error("application stopped due to error", slog.Any("error", err))
	}
}

// RunWithError lets a command report unexpected actor failures in its exit status.
func (r Runner) RunWithError() error {
	err := r.Group.Run(run.WithReverseShutdownOrder())
	if e := &(run.SignalError{}); errors.As(err, &e) {
		r.Logger.Info("received signal: shutting down", slog.String("signal", e.Signal.String()))
		return nil
	}

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

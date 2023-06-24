package gosundheit

import (
	health "github.com/AppsFlyer/go-sundheit"
	"golang.org/x/exp/slog"
)

type checkListener struct {
	logger *slog.Logger
}

func NewLogger(logger *slog.Logger) health.CheckListener {
	return checkListener{
		logger: logger,
	}
}

func (c checkListener) OnCheckRegistered(name string, result health.Result) {
	if result.Error != nil {
		c.logger.Error("initial check failed", slog.String("check", name), slog.Any("error", result.Error))

		return
	}

	c.logger.Info("check registered", slog.String("check", name))
}

func (c checkListener) OnCheckStarted(name string) {
	c.logger.Info("starting check", slog.String("check", name))
}

func (c checkListener) OnCheckCompleted(name string, result health.Result) {
	if result.Error != nil {
		c.logger.Error("check failed", slog.String("check", name), slog.Any("error", result.Error))

		return
	}

	c.logger.Info("check completed", slog.String("check", name))
}

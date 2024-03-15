package gosundheit

import (
	"fmt"
	"log/slog"

	health "github.com/AppsFlyer/go-sundheit"
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
		c.logger.Error(fmt.Errorf("initial health check failed; check: %s; error: %w", name, result.Error).Error())

		return
	}

	c.logger.Debug(fmt.Sprintf("health check registered: %s", name))
}

func (c checkListener) OnCheckStarted(name string) {
	c.logger.Debug(fmt.Sprintf("starting health check: %s", name))
}

func (c checkListener) OnCheckCompleted(name string, result health.Result) {
	if result.Error != nil {
		c.logger.Error(fmt.Errorf("health check failed; check: %s; error: %w", name, result.Error).Error())

		return
	}

	c.logger.Debug(fmt.Sprintf("health check completed: %s", name))
}

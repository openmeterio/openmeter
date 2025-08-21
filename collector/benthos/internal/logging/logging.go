package logging

import (
	"github.com/go-logr/logr"
	"github.com/redpanda-data/benthos/v4/public/service"
	"k8s.io/klog/v2"
)

// CtrlLogger adapts Benthos service.Logger to a logr.LogSink for controller-runtime and klog.
type CtrlLogger struct {
	logger *service.Logger
}

// Init is a no-op.
func (l *CtrlLogger) Init(info logr.RuntimeInfo) {}

// Enabled is a no-op.
func (l *CtrlLogger) Enabled(level int) bool {
	return true
}

// Info logs a message at the given level.
func (l *CtrlLogger) Info(level int, msg string, keysAndValues ...any) {
	if level == 0 {
		l.logger.Infof(msg, keysAndValues...)
		return
	}
	l.logger.Debugf(msg, keysAndValues...)
}

// Error logs an error message.
func (l *CtrlLogger) Error(err error, msg string, keysAndValues ...any) {
	l.logger.Errorf(msg, keysAndValues...)
}

// WithValues returns the logger. No-op.
func (l *CtrlLogger) WithValues(keysAndValues ...any) logr.LogSink {
	return l
}

// WithName returns the logger. No-op.
func (l *CtrlLogger) WithName(name string) logr.LogSink {
	return l
}

// NewLogrLogger returns a logr.Logger backed by Benthos logger.
func NewLogrLogger(logger *service.Logger) logr.Logger {
	return logr.New(&CtrlLogger{logger: logger})
}

// SetupKlog configures the global klog logger to use the provided Benthos logger via logr.
func SetupKlog(logger *service.Logger) {
	klog.SetLogger(NewLogrLogger(logger.With("component", "kubernetes")))
}

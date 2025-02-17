package common

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	health "github.com/AppsFlyer/go-sundheit"
	"github.com/oklog/run"

	"github.com/openmeterio/openmeter/app/config"
)

type TerminationState = string

const (
	TerminationStateRunning     TerminationState = "running"
	TerminationStateTerminating TerminationState = "terminating"
)

type status struct {
	State     TerminationState `json:"state"`
	Reason    error            `json:"reason,omitempty"`
	Timestamp time.Time        `json:"timestamp,omitempty"`
}

var _ health.Check = (*TerminationChecker)(nil)

type TerminationChecker struct {
	status

	mu sync.RWMutex

	propagationTimeout time.Duration
}

func (s *TerminationChecker) setDetails(state TerminationState, reason error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.State = state
	s.status.Reason = reason
	s.status.Timestamp = time.Now()
}

func (s *TerminationChecker) Terminate(reason error) {
	s.setDetails(TerminationStateTerminating, reason)
}

func (s *TerminationChecker) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.status.State == TerminationStateRunning
}

func (s *TerminationChecker) IsTerminating() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.status.State == TerminationStateTerminating
}

func (s *TerminationChecker) Name() string {
	return "termination.check"
}

func (s *TerminationChecker) Execute(_ context.Context) (interface{}, error) {
	details := struct {
		State     TerminationState `json:"state"`
		Reason    error            `json:"reason,omitempty"`
		Timestamp string           `json:"timestamp"`
	}{
		State:     s.status.State,
		Reason:    s.status.Reason,
		Timestamp: s.status.Timestamp.UTC().Format(time.RFC3339),
	}

	if s.IsTerminating() {
		return details, fmt.Errorf("termination was initiated [state=%s time=%s]: %w", details.State, details.Timestamp, details.Reason)
	}

	return details, nil
}

// WaitForPropagation blocks for
func (s *TerminationChecker) WaitForPropagation(ctx context.Context) error {
	if s.IsRunning() {
		return nil
	}

	propagationDeadline := s.status.Timestamp.Add(s.propagationTimeout)
	if time.Now().After(propagationDeadline) {
		return nil
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			if now.After(propagationDeadline) {
				return nil
			}
		}
	}
}

// NewTerminationChecker returns a new TerminationChecker and registers the TerminationChecker to
// the provided health checker.
func NewTerminationChecker(conf config.TerminationConfig, healthChecker health.Health) (*TerminationChecker, error) {
	rc := &TerminationChecker{
		status: status{
			State:     TerminationStateRunning,
			Timestamp: time.Now(),
		},
		propagationTimeout: conf.PropagationTimeout,
	}

	err := healthChecker.RegisterCheck(
		rc,
		health.ExecutionPeriod(conf.CheckInterval),
	)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

// NewTerminationCheckerActor returns actor functions (execute, interrupt) which can be passed to run.Add() to register it.
// Its state is set to TerminationStateStopping as soon as interrupt function is called.
func NewTerminationCheckerActor(r *TerminationChecker, logger *slog.Logger) (execute func() error, interrupt func(error), err error) {
	if r == nil {
		return nil, nil, errors.New("the TerminationChecker must not be nil")
	}

	if logger == nil {
		logger = slog.Default()
	}

	c := make(chan struct{})

	return func() error {
			// Wait until interrupt is invoked and return after
			<-c

			return nil
		}, func(err error) {
			logger.Debug("shutting down termination checker", "cause", err)

			r.Terminate(err)

			if errors.Is(err, run.ErrSignal) {
				logger.Debug("waiting for propagation of termination...", "until", r.Timestamp.Add(r.propagationTimeout))

				// Wait for propagation timeout
				if err = r.WaitForPropagation(context.Background()); err != nil {
					logger.Error("failed to wait for propagation", "error", err)
				}

				logger.Debug("waiting for propagation of termination is done", "until", r.Timestamp.Add(r.propagationTimeout))
			}

			close(c)
		}, nil
}

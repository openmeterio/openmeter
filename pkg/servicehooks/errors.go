package servicehooks

import (
	"errors"
	"fmt"
)

var (
	ErrContextRequired       = errors.New("context is required")
	ErrCycleDetected         = errors.New("service hook invocation cycle detected")
	ErrCyclePolicyInvalid    = errors.New("service hook cycle policy is invalid")
	ErrHookAlreadyRegistered = errors.New("service hook is already registered")
	ErrHookNameRequired      = errors.New("service hook name is required")
	ErrHookRequired          = errors.New("service hook is required")
	ErrPriorityOutOfRange    = errors.New("service hook priority is out of range")
	ErrRegisterOptionInvalid = errors.New("service hook register option is invalid")
	ErrRegistrySealed        = errors.New("service hook registry is sealed")
)

// CycleError identifies the active registration which caused a nested
// invocation to be rejected.
type CycleError struct {
	HookName string
	Priority Priority
}

func (e CycleError) Error() string {
	return fmt.Sprintf("%s: hook=%q priority=%d", ErrCycleDetected, e.HookName, e.Priority)
}

func (e CycleError) Unwrap() error {
	return ErrCycleDetected
}

// InvocationError adds registration identity and priority to a hook error.
type InvocationError struct {
	HookName string
	Priority Priority
	Err      error
}

func (e InvocationError) Error() string {
	return fmt.Sprintf("service hook failed: hook=%q priority=%d: %v", e.HookName, e.Priority, e.Err)
}

func (e InvocationError) Unwrap() error {
	return e.Err
}

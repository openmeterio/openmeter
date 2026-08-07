package servicehooks

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRegistryInvokesHooksByPriorityAndRegistrationOrder(t *testing.T) {
	var registry Registry[string]
	var invoked []string

	registrations := []struct {
		name     string
		priority Priority
	}{
		{name: "default-first", priority: PriorityDefault},
		{name: "lowest", priority: PriorityLowest},
		{name: "highest", priority: PriorityHighest},
		{name: "default-second", priority: PriorityDefault},
		{name: "high", priority: PriorityHigh},
	}

	for _, registration := range registrations {
		name := registration.name
		err := registry.Register(name, HookFunc[string](func(_ context.Context, event string) error {
			invoked = append(invoked, name+":"+event)

			return nil
		}), WithPriority(registration.priority))
		if err != nil {
			t.Fatalf("registering hook %q: %v", name, err)
		}
	}

	if err := registry.Invoke(t.Context(), "event"); err != nil {
		t.Fatalf("invoking hooks: %v", err)
	}

	expected := []string{
		"highest:event",
		"high:event",
		"default-first:event",
		"default-second:event",
		"lowest:event",
	}
	if len(invoked) != len(expected) {
		t.Fatalf("unexpected invocation count: got %d, expected %d", len(invoked), len(expected))
	}

	for i := range expected {
		if invoked[i] != expected[i] {
			t.Errorf("unexpected hook at index %d: got %q, expected %q", i, invoked[i], expected[i])
		}
	}
}

func TestRegistryUsesDefaultPriority(t *testing.T) {
	var registry Registry[struct{}]
	boom := errors.New("boom")

	if err := registry.Register("default", HookFunc[struct{}](func(context.Context, struct{}) error {
		return boom
	})); err != nil {
		t.Fatalf("registering hook: %v", err)
	}

	err := registry.Invoke(t.Context(), struct{}{})
	if !errors.Is(err, boom) {
		t.Fatalf("expected hook error, got %v", err)
	}

	var invocationError InvocationError
	if !errors.As(err, &invocationError) {
		t.Fatalf("expected InvocationError, got %T", err)
	}

	if invocationError.Priority != PriorityDefault {
		t.Errorf("unexpected priority: got %d, expected %d", invocationError.Priority, PriorityDefault)
	}
}

func TestRegistryStopsAtFirstHookError(t *testing.T) {
	var registry Registry[struct{}]
	boom := errors.New("boom")
	var secondInvoked bool

	if err := registry.Register("first", HookFunc[struct{}](func(context.Context, struct{}) error {
		return boom
	}), WithPriority(PriorityHigh)); err != nil {
		t.Fatalf("registering first hook: %v", err)
	}

	if err := registry.Register("second", HookFunc[struct{}](func(context.Context, struct{}) error {
		secondInvoked = true

		return nil
	})); err != nil {
		t.Fatalf("registering second hook: %v", err)
	}

	err := registry.Invoke(t.Context(), struct{}{})
	if !errors.Is(err, boom) {
		t.Fatalf("expected hook error, got %v", err)
	}
	if secondInvoked {
		t.Error("expected invocation to stop before the second hook")
	}

	var invocationError InvocationError
	if !errors.As(err, &invocationError) {
		t.Fatalf("expected InvocationError, got %T", err)
	}
	if invocationError.HookName != "first" {
		t.Errorf("unexpected hook name: got %q, expected %q", invocationError.HookName, "first")
	}
}

func TestRegistryRejectsInvalidRegistration(t *testing.T) {
	t.Run("name is required", func(t *testing.T) {
		var registry Registry[struct{}]
		err := registry.Register("  ", HookFunc[struct{}](func(context.Context, struct{}) error { return nil }))
		if !errors.Is(err, ErrHookNameRequired) {
			t.Fatalf("expected ErrHookNameRequired, got %v", err)
		}
	})

	t.Run("hook is required", func(t *testing.T) {
		var registry Registry[struct{}]
		var hook HookFunc[struct{}]
		err := registry.Register("nil", hook)
		if !errors.Is(err, ErrHookRequired) {
			t.Fatalf("expected ErrHookRequired, got %v", err)
		}
	})

	t.Run("hook name is unique after normalization", func(t *testing.T) {
		var registry Registry[struct{}]
		hook := HookFunc[struct{}](func(context.Context, struct{}) error { return nil })
		if err := registry.Register("hook", hook); err != nil {
			t.Fatalf("registering first hook: %v", err)
		}

		err := registry.Register(" hook ", hook)
		if !errors.Is(err, ErrHookAlreadyRegistered) {
			t.Fatalf("expected ErrHookAlreadyRegistered, got %v", err)
		}
	})

	t.Run("priority is in range", func(t *testing.T) {
		for _, priority := range []Priority{-1, 101} {
			var registry Registry[struct{}]
			err := registry.Register(
				"invalid-priority",
				HookFunc[struct{}](func(context.Context, struct{}) error { return nil }),
				WithPriority(priority),
			)
			if !errors.Is(err, ErrPriorityOutOfRange) {
				t.Fatalf("expected ErrPriorityOutOfRange for %d, got %v", priority, err)
			}
			if !errors.Is(err, ErrRegisterOptionInvalid) {
				t.Fatalf("expected ErrRegisterOptionInvalid for %d, got %v", priority, err)
			}
		}
	})

	t.Run("cycle policy is valid", func(t *testing.T) {
		var registry Registry[struct{}]
		err := registry.Register(
			"invalid-cycle-policy",
			HookFunc[struct{}](func(context.Context, struct{}) error { return nil }),
			WithCyclePolicy(CyclePolicy(100)),
		)
		if !errors.Is(err, ErrCyclePolicyInvalid) {
			t.Fatalf("expected ErrCyclePolicyInvalid, got %v", err)
		}
	})

	t.Run("option is not nil", func(t *testing.T) {
		var registry Registry[struct{}]
		var option RegisterOption
		err := registry.Register(
			"nil-option",
			HookFunc[struct{}](func(context.Context, struct{}) error { return nil }),
			option,
		)
		if !errors.Is(err, ErrRegisterOptionInvalid) {
			t.Fatalf("expected ErrRegisterOptionInvalid, got %v", err)
		}
	})
}

func TestRegistrySealsExplicitlyOrOnFirstInvocation(t *testing.T) {
	t.Run("explicit seal", func(t *testing.T) {
		var registry Registry[struct{}]
		registry.Seal()
		if !registry.IsSealed() {
			t.Fatal("expected registry to be sealed")
		}

		err := registry.Register("late", HookFunc[struct{}](func(context.Context, struct{}) error { return nil }))
		if !errors.Is(err, ErrRegistrySealed) {
			t.Fatalf("expected ErrRegistrySealed, got %v", err)
		}
	})

	t.Run("first invocation", func(t *testing.T) {
		var registry Registry[struct{}]
		if err := registry.Invoke(t.Context(), struct{}{}); err != nil {
			t.Fatalf("invoking empty registry: %v", err)
		}
		if !registry.IsSealed() {
			t.Fatal("expected registry to be sealed")
		}

		err := registry.Register("late", HookFunc[struct{}](func(context.Context, struct{}) error { return nil }))
		if !errors.Is(err, ErrRegistrySealed) {
			t.Fatalf("expected ErrRegistrySealed, got %v", err)
		}
	})
}

func TestRegistryReportsCyclesByDefaultBeforeNestedHooksRun(t *testing.T) {
	var registry Registry[int]
	var recursiveCalls int
	var tailCalls int

	if err := registry.Register("recursive", HookFunc[int](func(ctx context.Context, event int) error {
		recursiveCalls++

		return registry.Invoke(ctx, event+1)
	}), WithPriority(PriorityHighest)); err != nil {
		t.Fatalf("registering recursive hook: %v", err)
	}

	if err := registry.Register("tail", HookFunc[int](func(context.Context, int) error {
		tailCalls++

		return nil
	})); err != nil {
		t.Fatalf("registering tail hook: %v", err)
	}

	err := registry.Invoke(t.Context(), 0)
	if !errors.Is(err, ErrCycleDetected) {
		t.Fatalf("expected ErrCycleDetected, got %v", err)
	}

	var cycleError CycleError
	if !errors.As(err, &cycleError) {
		t.Fatalf("expected CycleError, got %T", err)
	}
	if cycleError.HookName != "recursive" {
		t.Errorf("unexpected cyclic hook: got %q, expected %q", cycleError.HookName, "recursive")
	}
	if recursiveCalls != 1 {
		t.Errorf("unexpected recursive hook calls: got %d, expected 1", recursiveCalls)
	}
	if tailCalls != 0 {
		t.Errorf("nested invocation ran hooks before reporting the cycle: tail calls=%d", tailCalls)
	}
}

func TestRegistryCanSkipOnlyTheActiveRegistration(t *testing.T) {
	var registry Registry[int]
	var recursiveCalls int
	var tailCalls int

	if err := registry.Register("recursive", HookFunc[int](func(ctx context.Context, event int) error {
		recursiveCalls++
		if event == 0 {
			return registry.Invoke(ctx, event+1)
		}

		return nil
	}), WithPriority(PriorityHighest), WithCyclePolicy(CyclePolicySkip)); err != nil {
		t.Fatalf("registering recursive hook: %v", err)
	}

	if err := registry.Register("tail", HookFunc[int](func(context.Context, int) error {
		tailCalls++

		return nil
	})); err != nil {
		t.Fatalf("registering tail hook: %v", err)
	}

	if err := registry.Invoke(t.Context(), 0); err != nil {
		t.Fatalf("invoking hooks: %v", err)
	}
	if recursiveCalls != 1 {
		t.Errorf("unexpected recursive hook calls: got %d, expected 1", recursiveCalls)
	}
	if tailCalls != 2 {
		t.Errorf("expected tail hook in nested and outer invocation: got %d calls", tailCalls)
	}
}

func TestRegistryDeactivatesCycleFrameAfterHookReturns(t *testing.T) {
	var registry Registry[int]
	var retained context.Context
	var calls int

	if err := registry.Register("capture-context", HookFunc[int](func(ctx context.Context, _ int) error {
		calls++
		retained = ctx

		return nil
	})); err != nil {
		t.Fatalf("registering hook: %v", err)
	}

	if err := registry.Invoke(t.Context(), 1); err != nil {
		t.Fatalf("first invocation: %v", err)
	}
	if err := registry.Invoke(retained, 2); err != nil {
		t.Fatalf("invocation with retained context: %v", err)
	}
	if calls != 2 {
		t.Errorf("unexpected hook calls: got %d, expected 2", calls)
	}
}

func TestRegistryDoesNotHoldLockWhileInvokingHook(t *testing.T) {
	var registry Registry[struct{}]
	var registrationErr error

	if err := registry.Register("register-late", HookFunc[struct{}](func(context.Context, struct{}) error {
		registrationErr = registry.Register("late", HookFunc[struct{}](func(context.Context, struct{}) error { return nil }))

		return nil
	})); err != nil {
		t.Fatalf("registering hook: %v", err)
	}

	if err := registry.Invoke(t.Context(), struct{}{}); err != nil {
		t.Fatalf("invoking hook: %v", err)
	}
	if !errors.Is(registrationErr, ErrRegistrySealed) {
		t.Fatalf("expected late registration to return ErrRegistrySealed, got %v", registrationErr)
	}
}

func TestRegistryHonorsContextCancellation(t *testing.T) {
	var registry Registry[struct{}]
	var invoked bool

	if err := registry.Register("hook", HookFunc[struct{}](func(context.Context, struct{}) error {
		invoked = true

		return nil
	})); err != nil {
		t.Fatalf("registering hook: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := registry.Invoke(ctx, struct{}{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if invoked {
		t.Error("hook ran after context cancellation")
	}
}

func TestRegistryStopsWhenContextIsCanceledBetweenHooks(t *testing.T) {
	var registry Registry[struct{}]
	ctx, cancel := context.WithCancel(t.Context())
	var secondInvoked bool

	if err := registry.Register("cancel", HookFunc[struct{}](func(context.Context, struct{}) error {
		cancel()

		return nil
	})); err != nil {
		t.Fatalf("registering cancel hook: %v", err)
	}

	if err := registry.Register("second", HookFunc[struct{}](func(context.Context, struct{}) error {
		secondInvoked = true

		return nil
	})); err != nil {
		t.Fatalf("registering second hook: %v", err)
	}

	err := registry.Invoke(ctx, struct{}{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if secondInvoked {
		t.Error("second hook ran after context cancellation")
	}
}

func TestRegistryRejectsNilContext(t *testing.T) {
	var registry Registry[struct{}]

	// NOTE: nil context.Context is deliberately tested here
	//nolint:staticcheck
	err := registry.Invoke(nil, struct{}{})
	if !errors.Is(err, ErrContextRequired) {
		t.Fatalf("expected ErrContextRequired, got %v", err)
	}
	if registry.IsSealed() {
		t.Error("invalid invocation sealed the registry")
	}
}

func TestRegistrySupportsConcurrentInvocation(t *testing.T) {
	var registry Registry[int]
	var calls atomic.Int64

	if err := registry.Register("counter", HookFunc[int](func(context.Context, int) error {
		calls.Add(1)

		return nil
	})); err != nil {
		t.Fatalf("registering hook: %v", err)
	}

	const invocationCount = 100
	ctx := t.Context()
	var wg sync.WaitGroup
	errs := make(chan error, invocationCount)
	for i := 0; i < invocationCount; i++ {
		wg.Add(1)
		go func(event int) {
			defer wg.Done()

			errs <- registry.Invoke(ctx, event)
		}(i)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent invocation failed: %v", err)
		}
	}

	if calls.Load() != invocationCount {
		t.Errorf("unexpected hook calls: got %d, expected %d", calls.Load(), invocationCount)
	}
}

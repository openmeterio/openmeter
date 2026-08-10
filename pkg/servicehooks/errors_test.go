package servicehooks

import "testing"

func TestInvocationErrorFormatsNilCause(t *testing.T) {
	err := InvocationError{HookName: "hook"}

	const expected = "service hook failed [service.hook=hook]: <nil>"
	if got := err.Error(); got != expected {
		t.Errorf("unexpected error: got %q, expected %q", got, expected)
	}
}

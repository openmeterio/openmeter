package softdelete

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkip(t *testing.T) {
	ctx := context.Background()
	assert.False(t, IsSkipped(ctx))

	skipped := Skip(ctx)
	assert.True(t, IsSkipped(skipped))
	// Original ctx is untouched.
	assert.False(t, IsSkipped(ctx))
}

func TestAllowHardDelete(t *testing.T) {
	ctx := context.Background()
	assert.False(t, IsHardDeleteAllowed(ctx))

	allowed := AllowHardDelete(ctx)
	assert.True(t, IsHardDeleteAllowed(allowed))
	// Skip and AllowHardDelete are independent context flags.
	assert.False(t, IsSkipped(allowed))
	// Original ctx is untouched.
	assert.False(t, IsHardDeleteAllowed(ctx))
}

// TestActivePredicate_SQL verifies the predicate emits the expected
// time-windowed clause. We render the selector to SQL and assert on the
// resulting fragment because the predicate's only contract is what it
// generates.
func TestActivePredicate_SQL(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	s := sql.Dialect(dialect.Postgres).Select("*").From(sql.Table("rows"))
	ActivePredicate(now)(s)

	query, args := s.Query()
	assert.Contains(t, query, `"deleted_at" IS NULL`)
	assert.Contains(t, query, `"deleted_at" > `)
	require.Len(t, args, 1)
	assert.Equal(t, now, args[0])
}

// TestRegisterRunCascadeFor exercises the registry round trip and the
// duplicate-registration guard via the simpler RunCascadeFor entry point
// that does not require a mutation.
func TestRegisterRunCascadeFor(t *testing.T) {
	cascadeMu.Lock()
	prev := cascadeRegistry
	cascadeRegistry = map[string]CascadeFunc{}
	cascadeMu.Unlock()
	t.Cleanup(func() {
		cascadeMu.Lock()
		cascadeRegistry = prev
		cascadeMu.Unlock()
	})

	var (
		gotIDs    []any
		gotClient any
		callsCnt  int
	)
	Register("Test", func(ctx context.Context, client any, ids []any) error {
		callsCnt++
		gotIDs = append([]any(nil), ids...)
		gotClient = client
		return nil
	})

	sentinel := struct{ name string }{name: "client-sentinel"}
	require.NoError(t, RunCascadeFor(t.Context(), "Test", sentinel, []any{"a", "b"}))
	assert.Equal(t, 1, callsCnt)
	assert.Equal(t, []any{"a", "b"}, gotIDs)
	assert.Equal(t, sentinel, gotClient)

	// Empty ID set short-circuits.
	require.NoError(t, RunCascadeFor(t.Context(), "Test", sentinel, nil))
	assert.Equal(t, 1, callsCnt)

	// Unregistered type is a no-op.
	require.NoError(t, RunCascadeFor(t.Context(), "Other", sentinel, []any{"x"}))
	assert.Equal(t, 1, callsCnt)

	// Duplicate registration panics.
	assert.Panics(t, func() {
		Register("Test", func(context.Context, any, []any) error { return nil })
	})

	// Nil walker is silently dropped.
	Register("Nil", nil)
	require.NoError(t, RunCascadeFor(t.Context(), "Nil", sentinel, []any{"x"}))
}

// TestRunCascade_NoWalker verifies the mutation-driven entry point is a
// no-op when no walker is registered for the mutation's type. This is the
// path the soft-delete hook hits for entities that have no outgoing
// soft-delete edges.
func TestRunCascade_NoWalker(t *testing.T) {
	cascadeMu.Lock()
	prev := cascadeRegistry
	cascadeRegistry = map[string]CascadeFunc{}
	cascadeMu.Unlock()
	t.Cleanup(func() {
		cascadeMu.Lock()
		cascadeRegistry = prev
		cascadeMu.Unlock()
	})

	// Empty IDs short-circuits before reflection.
	require.NoError(t, RunCascade(t.Context(), stubMutation{typ: "Test"}, nil))

	// Non-empty IDs but no registered walker also short-circuits before
	// trying to extract a Client() from the mutation (which our stub
	// doesn't have).
	require.NoError(t, RunCascade(t.Context(), stubMutation{typ: "Test"}, []string{"a"}))
}

// TestInterceptor_SkipPasses verifies the interceptor short-circuits on
// Skip(ctx) without inspecting the query, which would otherwise fail the
// type assertion in tests that do not pass a real ent query.
func TestInterceptor_SkipPasses(t *testing.T) {
	i := Interceptor()
	tf, ok := i.(ent.TraverseFunc)
	require.True(t, ok)

	err := tf(Skip(t.Context()), nil) // nil query is fine because Skip short-circuits.
	require.NoError(t, err)
}

// stubMutation implements ent.Mutation just enough for RunCascade.
type stubMutation struct {
	typ string
}

func (s stubMutation) Op() ent.Op {
	return 0
}

func (s stubMutation) Type() string {
	return s.typ
}

func (s stubMutation) Fields() []string {
	return nil
}

func (s stubMutation) Field(string) (ent.Value, bool) {
	return nil, false
}

func (s stubMutation) OldField(context.Context, string) (ent.Value, error) {
	return nil, nil
}

func (s stubMutation) SetField(string, ent.Value) error {
	return nil
}

func (s stubMutation) AddedFields() []string {
	return nil
}

func (s stubMutation) AddedField(string) (ent.Value, bool) {
	return nil, false
}

func (s stubMutation) AddField(string, ent.Value) error {
	return nil
}

func (s stubMutation) ClearedFields() []string {
	return nil
}

func (s stubMutation) FieldCleared(string) bool {
	return false
}

func (s stubMutation) ClearField(string) error {
	return nil
}

func (s stubMutation) ResetField(string) error {
	return nil
}

func (s stubMutation) AddedEdges() []string {
	return nil
}

func (s stubMutation) AddedIDs(string) []ent.Value {
	return nil
}

func (s stubMutation) RemovedEdges() []string {
	return nil
}

func (s stubMutation) RemovedIDs(string) []ent.Value {
	return nil
}

func (s stubMutation) ClearedEdges() []string {
	return nil
}

func (s stubMutation) EdgeCleared(string) bool {
	return false
}

func (s stubMutation) ClearEdge(string) error {
	return nil
}

func (s stubMutation) ResetEdge(string) error {
	return nil
}

func (s stubMutation) Where(...func(*sql.Selector)) {}

func (s stubMutation) WhereP(...func(*sql.Selector)) {}

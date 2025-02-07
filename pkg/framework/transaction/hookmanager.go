package transaction

import (
	"context"
	"errors"
	"log/slog"
)

// Hook is the external type for hooks
type Hook func(ctx context.Context) error

// hook swallows the errors and logs instead to not to break the execution chain
// as we are using hooks for message publishing only, and we have countermeasures for
// missing events (e.g. periodic reconciliation jobs)
type hook func(ctx context.Context)

func loggingHook(h Hook) hook {
	return func(ctx context.Context) {
		if err := h(ctx); err != nil {
			slog.ErrorContext(ctx, "error executing post commit hook", "error", err)
		}
	}
}

type HookLayers [][]hook

// AppendToLastLayer appends hooks to the last layer of hooks
func (h *HookLayers) AppendToLastLayer(hooks ...hook) bool {
	if len(*h) == 0 {
		return false
	}

	(*h)[len(*h)-1] = append((*h)[len(*h)-1], hooks...)
	return true
}

// DiscardLastLayer discards the last layer of hooks, return false if there are no layers to discard
func (h *HookLayers) DiscardLastLayer() bool {
	if len(*h) == 0 {
		return false
	}

	*h = (*h)[:len(*h)-1]
	return true
}

// GetLastLayer returns the last layer of hooks
func (h HookLayers) GetLastLayer() ([]hook, bool) {
	if len(h) == 0 {
		return nil, false
	}

	return h[len(h)-1], true
}

// AddLayer adds a new layer of hooks
func (h *HookLayers) AddLayer() {
	*h = append(*h, []hook{})
}

// GetLayerCount returns the number of layers
func (h HookLayers) GetLayerCount() int {
	return len(h)
}

type TransactionHooks struct {
	PostCommitHooks HookLayers
}

func (t *TransactionHooks) AddBeforeCommitHook(hook hook) error {
	ok := t.PostCommitHooks.AppendToLastLayer(hook)

	if !ok {
		// This signals that we have never called SavePoint
		return errors.New("hook logic error: no hooks to append to")
	}

	return nil
}

func (t *TransactionHooks) PostSavePoint() {
	t.PostCommitHooks.AddLayer()
}

func (t *TransactionHooks) PostCommit(ctx context.Context) error {
	layerCount := t.PostCommitHooks.GetLayerCount()

	if layerCount == 0 {
		return errors.New("hook logic error: there are no layers available in post commit hooks for commit")
	}

	if layerCount == 1 {
		// We are the last save point => let's execute the hooks
		finalLayer, _ := t.PostCommitHooks.GetLastLayer()
		for _, hook := range finalLayer {
			hook(ctx)
		}

		t.PostCommitHooks = HookLayers{}
		return nil
	}

	// We are not the last save point => let's add the commit hooks to the parent
	currentHooks, _ := t.PostCommitHooks.GetLastLayer()

	t.PostCommitHooks.AppendToLastLayer(currentHooks...)
	return nil
}

func (t *TransactionHooks) PostRollback() error {
	if t.PostCommitHooks.GetLayerCount() == 0 {
		return errors.New("hook logic error: there are no layers available in post commit hooks for rollback")
	}

	_ = t.PostCommitHooks.DiscardLastLayer()
	return nil
}

// Context handling

type omHookManagerContextKey string

const hookManagerContextKey omHookManagerContextKey = "hook_manager_context_key"

func GetHookManagerFromContext(ctx context.Context) (*TransactionHooks, error) {
	hooks, ok := ctx.Value(hookManagerContextKey).(*TransactionHooks)
	if !ok {
		return nil, &HookManagerNotFoundError{}
	}
	return hooks, nil
}

type HookManagerNotFoundError struct{}

func (e *HookManagerNotFoundError) Error() string {
	return "hook manager not found in context"
}

func SetHookManagerOnContext(ctx context.Context, hooks *TransactionHooks) (context.Context, error) {
	if _, err := GetHookManagerFromContext(ctx); err == nil {
		return ctx, &HookManagerConflictError{}
	}
	return context.WithValue(ctx, hookManagerContextKey, hooks), nil
}

type HookManagerConflictError struct{}

func (e *HookManagerConflictError) Error() string {
	return "hook manager already exists in context"
}

func UpsertHookManagerOnContext(ctx context.Context) context.Context {
	if _, err := GetHookManagerFromContext(ctx); err == nil {
		return ctx
	}

	hooks := &TransactionHooks{
		PostCommitHooks: HookLayers{},
	}
	ctx, _ = SetHookManagerOnContext(ctx, hooks)
	return ctx
}

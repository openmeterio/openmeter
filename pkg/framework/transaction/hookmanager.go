package transaction

import (
	"context"
	"errors"
	"log/slog"
	"sync"
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

type hookLayers [][]hook

// AppendToLastLayer appends hooks to the last layer of hooks
func (h *hookLayers) AppendToLastLayer(hooks ...hook) bool {
	if len(*h) == 0 {
		return false
	}

	(*h)[len(*h)-1] = append((*h)[len(*h)-1], hooks...)
	return true
}

// DiscardLastLayer discards the last layer of hooks, return false if there are no layers to discard
func (h *hookLayers) DiscardLastLayer() bool {
	if len(*h) == 0 {
		return false
	}

	*h = (*h)[:len(*h)-1]
	return true
}

// GetLastLayer returns the last layer of hooks
func (h hookLayers) GetLastLayer() ([]hook, bool) {
	if len(h) == 0 {
		return nil, false
	}

	return h[len(h)-1], true
}

// AddLayer adds a new layer of hooks
func (h *hookLayers) AddLayer() {
	*h = append(*h, []hook{})
}

// GetLayerCount returns the number of layers
func (h hookLayers) GetLayerCount() int {
	return len(h)
}

type hookManager struct {
	mu sync.Mutex

	PostCommitHooks hookLayers
}

func (m *hookManager) AddPostCommitHook(cb hook) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ok := m.PostCommitHooks.AppendToLastLayer(cb)

	if !ok {
		// This signals that we have never called SavePoint
		return errors.New("hook logic error: no hooks to append to")
	}

	return nil
}

func (m *hookManager) PostSavePoint() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PostCommitHooks.AddLayer()
}

func (m *hookManager) postCommit() ([]hook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	layerCount := m.PostCommitHooks.GetLayerCount()

	if layerCount == 0 {
		return nil, errors.New("hook logic error: there are no layers available in post commit hooks for commit")
	}

	if layerCount == 1 {
		// We are the last save point => let's execute the hooks
		finalLayer, _ := m.PostCommitHooks.GetLastLayer()

		m.PostCommitHooks = hookLayers{}
		return finalLayer, nil
	}

	// We are not the last save point => let's add the commit hooks to the parent
	currentHooks, _ := m.PostCommitHooks.GetLastLayer()

	m.PostCommitHooks.AppendToLastLayer(currentHooks...)
	return nil, nil
}

func (m *hookManager) PostCommit(ctx context.Context) error {
	hooks, err := m.postCommit()
	if err != nil {
		return err
	}

	for _, hook := range hooks {
		hook(ctx)
	}

	return nil
}

func (m *hookManager) PostRollback() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.PostCommitHooks.GetLayerCount() == 0 {
		return errors.New("hook logic error: there are no layers available in post commit hooks for rollback")
	}

	_ = m.PostCommitHooks.DiscardLastLayer()
	return nil
}

// Context handling

type omhookManagerContextKey string

const hookManagerContextKey omhookManagerContextKey = "hook_manager_context_key"

func getHookManagerFromContext(ctx context.Context) (*hookManager, error) {
	hooks, ok := ctx.Value(hookManagerContextKey).(*hookManager)
	if !ok {
		return nil, &hookManagerNotFoundError{}
	}
	return hooks, nil
}

type hookManagerNotFoundError struct{}

func (e *hookManagerNotFoundError) Error() string {
	return "hook manager not found in context"
}

func setHookManagerOnContext(ctx context.Context, hooks *hookManager) (context.Context, error) {
	if _, err := getHookManagerFromContext(ctx); err == nil {
		return ctx, &hookManagerConflictError{}
	}
	return context.WithValue(ctx, hookManagerContextKey, hooks), nil
}

type hookManagerConflictError struct{}

func (e *hookManagerConflictError) Error() string {
	return "hook manager already exists in context"
}

func upserthookManagerOnContext(ctx context.Context) (context.Context, *hookManager) {
	if hookMgr, err := getHookManagerFromContext(ctx); err == nil {
		return ctx, hookMgr
	}

	hookMgr := &hookManager{
		PostCommitHooks: hookLayers{},
	}
	ctx, _ = setHookManagerOnContext(ctx, hookMgr)
	return ctx, hookMgr
}

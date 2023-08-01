package namespace

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFakeHandler() *fakeHandler {
	return &fakeHandler{
		namespaces: map[string]bool{},
	}
}

type fakeHandler struct {
	namespaces map[string]bool

	mu sync.Mutex
}

func (h *fakeHandler) CreateNamespace(_ context.Context, name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.namespaces[name] = true

	return nil
}

func TestManager_CreateNamespce(t *testing.T) {
	handler := newFakeHandler()

	manager, err := NewManager(ManagerConfig{
		Handlers:         []Handler{handler},
		DefaultNamespace: "default",
	})
	require.NoError(t, err)

	const namespace = "my-namespace"

	err = manager.CreateNamespace(context.Background(), namespace)
	require.NoError(t, err)

	assert.True(t, handler.namespaces[namespace])
}

func TestManager_CreateDefaultNamespce(t *testing.T) {
	handler := newFakeHandler()

	manager, err := NewManager(ManagerConfig{
		Handlers:         []Handler{handler},
		DefaultNamespace: "default",
	})
	require.NoError(t, err)

	err = manager.CreateDefaultNamespace(context.Background())
	require.NoError(t, err)

	assert.True(t, handler.namespaces["default"])
}

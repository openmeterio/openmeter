// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

func (h *fakeHandler) DeleteNamespace(_ context.Context, name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.namespaces, name)

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

func TestManager_DeleteNamespce(t *testing.T) {
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

	err = manager.DeleteNamespace(context.Background(), namespace)
	require.NoError(t, err)

	assert.False(t, handler.namespaces[namespace])
}

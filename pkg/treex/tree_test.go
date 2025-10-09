package treex

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testVal struct{ id int }

func TestNewTree_NilRoot(t *testing.T) {
	tr, err := NewTree[*testVal](nil)
	require.Error(t, err)
	assert.Nil(t, tr)
	assert.ErrorIs(t, err, ErrRootNodeIsNil)
}

func TestNewTree_MutatesInput(t *testing.T) {
	// Graph:
	// 1
	// └── 2
	v := &testVal{id: 1}
	root := NewNode(v)

	childVal := &testVal{id: 2}
	child := NewNode(childVal)
	root.AddChild(child)

	origParent := child.Parent()
	origChildren := root.Children()

	tr, err := NewTree(root)
	require.NoError(t, err)
	require.NotNil(t, tr)

	// original nodes remain unchanged as we didn't mutat them
	assert.Equal(t, origParent, child.Parent())
	assert.Equal(t, origChildren, root.Children())
	assert.NotNil(t, tr.root)
	// ensure root pointer is the same (can be mutated)
	assert.Same(t, root, tr.root)
}

func TestNewTree_DontChangeRoot(t *testing.T) {
	// Graph:
	// 1
	// └── 2
	//     └── 3
	v1 := &testVal{id: 1}
	v2 := &testVal{id: 2}
	v3 := &testVal{id: 3}

	grand := NewNode(v1)
	parent := NewNode(v2)
	child := NewNode(v3)

	grand.AddChild(parent)
	parent.AddChild(child)

	// Build tree starting from parent (non-root)
	tr, err := NewTree(parent)
	require.NoError(t, err)
	require.NotNil(t, tr)

	// It should expose the same children as the provided parent
	require.Len(t, tr.root.Children(), 1)
	assert.Equal(t, child, tr.root.Children()[0])
	// Provided parent remains with its original parent
	assert.Equal(t, grand, parent.Parent())
	// Pointer identity is the same
	assert.Same(t, parent, tr.root)
}

func TestNewTree_AcyclicValid(t *testing.T) {
	// Graph:
	// 1
	// ├── 2
	// └── 3
	v1 := &testVal{id: 1}
	v2 := &testVal{id: 2}
	v3 := &testVal{id: 3}

	r := NewNode(v1)
	c1 := NewNode(v2)
	c2 := NewNode(v3)

	r.AddChild(c1)
	r.AddChild(c2)

	tr, err := NewTree(r)
	require.NoError(t, err)
	require.NotNil(t, tr)
}

func TestNewTree_DetectsCycle(t *testing.T) {
	// Graph (cycle):
	// 1 -> 2 -> 1
	v1 := &testVal{id: 1}
	v2 := &testVal{id: 2}

	n1 := NewNode(v1)
	n2 := NewNode(v2)

	n1.AddChild(n2)
	// introduce a cycle by making n2 a parent of n1
	n2.AddChild(n1)

	tr, err := NewTree(n1)
	require.Error(t, err)
	assert.Nil(t, tr)
	assert.ErrorIs(t, err, ErrGraphHasCycle)
}

func TestNewTree_InvalidGraph_NilChild(t *testing.T) {
	// Graph (invalid):
	// 1
	// └── <nil>
	v1 := &testVal{id: 1}

	r := NewNode(v1)
	// Corrupt structure: append a nil child directly
	r.children = append(r.children, nil)

	tr, err := NewTree(r)
	require.Error(t, err)
	assert.Nil(t, tr)
	assert.ErrorIs(t, err, ErrNodeGraphInvalid)
}

func TestDFS_PreOrderTraversal(t *testing.T) {
	// Graph:
	// 1
	// ├── 2
	// │   └── 4
	// └── 3
	v1 := &testVal{id: 1}
	v2 := &testVal{id: 2}
	v3 := &testVal{id: 3}
	v4 := &testVal{id: 4}

	r := NewNode(v1)
	c1 := NewNode(v2)
	c2 := NewNode(v3)
	c1a := NewNode(v4)

	r.AddChild(c1)
	r.AddChild(c2)
	c1.AddChild(c1a)

	trr, err := NewTree(r)
	require.NoError(t, err)

	var visited []int
	err = trr.DFS(func(n *Node[*testVal]) (bool, error) {
		visited = append(visited, n.Value().id)
		return false, nil
	})
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2, 4, 3}, visited, "preorder traversal expected")
}

func TestDFS_PruneSubtreeOnStop(t *testing.T) {
	// Graph:
	// 1
	// ├── 2
	// │   └── 4
	// └── 3
	v1 := &testVal{id: 1}
	v2 := &testVal{id: 2}
	v3 := &testVal{id: 3}
	v4 := &testVal{id: 4}

	r := NewNode(v1)
	c1 := NewNode(v2)
	c2 := NewNode(v3)
	c1a := NewNode(v4)

	r.AddChild(c1)
	r.AddChild(c2)
	c1.AddChild(c1a)

	trr, err := NewTree(r)
	require.NoError(t, err)

	var visited []int
	err = trr.DFS(func(n *Node[*testVal]) (bool, error) {
		visited = append(visited, n.Value().id)
		if n.Value().id == 2 {
			return true, nil // prune children of 2 (i.e., node 4)
		}
		return false, nil
	})
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2, 3}, visited, "should prune 2's subtree and continue with siblings")
}

func TestDFS_ErrorPropagation(t *testing.T) {
	// Graph:
	// 1
	// ├── 2
	// │   └── 4
	// └── 3
	//     └── 5
	v1 := &testVal{id: 1}
	v2 := &testVal{id: 2}
	v3 := &testVal{id: 3}
	v4 := &testVal{id: 4}
	v5 := &testVal{id: 5}

	r := NewNode(v1)
	c1 := NewNode(v2)
	c2 := NewNode(v3)
	c1a := NewNode(v4)
	c2a := NewNode(v5)

	r.AddChild(c1)
	r.AddChild(c2)
	c1.AddChild(c1a)
	c2.AddChild(c2a)

	trr, err := NewTree(r)
	require.NoError(t, err)

	var visited []int
	boom := errors.New("boom")
	err = trr.DFS(func(n *Node[*testVal]) (bool, error) {
		visited = append(visited, n.Value().id)
		if n.Value().id == 3 {
			return false, boom
		}
		return false, nil
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	// Should not have visited 5 (child of 3) after error
	assert.Equal(t, []int{1, 2, 4, 3}, visited)
}

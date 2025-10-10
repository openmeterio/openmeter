package treex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wrapper type that owns a value and has a Node pointing to itself
// This mirrors patterns like FieldDescriptor which maintain a Node[*T]
// to participate in a larger tree.
type wrapper struct {
	name string
	node *Node[*wrapper]
}

func newWrapper(name string) *wrapper {
	w := &wrapper{name: name}
	w.node = NewNode(w)
	return w
}

func TestWrapper_EmbeddingAndHierarchy(t *testing.T) {
	// Graph:
	// root
	// ├── a
	// │   └── a1
	// └── b
	root := newWrapper("root")
	a := newWrapper("a")
	a1 := newWrapper("a1")
	b := newWrapper("b")

	root.node.AddChild(a.node)
	a.node.AddChild(a1.node)
	root.node.AddChild(b.node)

	tr, err := NewTree(root.node)
	require.NoError(t, err)
	require.NotNil(t, tr)

	var order []string
	require.NoError(t, tr.DFS(func(n *Node[*wrapper]) (bool, error) {
		order = append(order, n.Value().name)
		return false, nil
	}))

	assert.Equal(t, []string{"root", "a", "a1", "b"}, order)
	// NewTree shallow-clones root and reattaches first-level children to the clone
	assert.Equal(t, tr.Root(), a.node.Parent())
	assert.True(t, a1.node.Parent() == a.node)
}

func TestWrapper_ImmutableUpdate_UsingShallowClone(t *testing.T) {
	// We want to "rename" the wrapper at the root without mutating the original
	root := newWrapper("root")
	child := newWrapper("child")
	root.node.AddChild(child.node)

	// Build an initial tree
	tr, err := NewTree(root.node)
	require.NoError(t, err)

	// To perform an immutable-style update on the root wrapper object,
	// we create a new wrapper copy, and we shallow-clone the node to keep the
	// existing relationships (children) but swap to the new value.
	updated := &wrapper{name: "root-renamed"}
	updated.node = root.node.ShallowClone()
	updated.node.SetValue(updated)
	// ensure node identity differs from the original root.node (clone)
	assert.NotSame(t, root.node, updated.node)
	// children and parent links preserved by the shallow clone
	require.Len(t, updated.node.Children(), 1)
	assert.Equal(t, child.node, updated.node.Children()[0])
	assert.Nil(t, updated.node.Parent())

	// Build a tree from the updated node; original tree remains intact
	tr2, err := NewTree(updated.node)
	require.NoError(t, err)

	var order1 []string
	require.NoError(t, tr.DFS(func(n *Node[*wrapper]) (bool, error) {
		order1 = append(order1, n.Value().name)
		return false, nil
	}))
	assert.Equal(t, []string{"root", "child"}, order1)

	var order2 []string
	require.NoError(t, tr2.DFS(func(n *Node[*wrapper]) (bool, error) {
		order2 = append(order2, n.Value().name)
		return false, nil
	}))
	assert.Equal(t, []string{"root-renamed", "child"}, order2)
}

func TestWrapper_PartialSubtreeUpdate_WithShallowCloneAtIntermediate(t *testing.T) {
	// Graph initial:
	// root
	// ├── left
	// │   └── ll
	// └── right
	root := newWrapper("root")
	left := newWrapper("left")
	right := newWrapper("right")
	ll := newWrapper("ll")

	root.node.AddChild(left.node)
	root.node.AddChild(right.node)
	left.node.AddChild(ll.node)

	// We want to update only the 'left' wrapper (e.g., rename or add metadata)
	// without touching other branches. Clone the left node shallowly and swap it.
	leftUpdated := &wrapper{name: "left*"}
	leftUpdated.node = left.node.ShallowClone()
	leftUpdated.node.SetValue(leftUpdated)

	// ShallowClone reattaches first-level children, so no manual reattachment is needed.

	// Swap the child on root from left.node to leftUpdated.node
	require.NoError(t, root.node.SwapChild(left.node, leftUpdated.node))
	assert.Equal(t, root.node, leftUpdated.node.Parent())
	// original left node is now detached
	assert.Nil(t, left.node.Parent())
	// leftUpdated still has original child's identity (ll)
	require.Len(t, leftUpdated.node.Children(), 1)
	assert.Equal(t, ll.node, leftUpdated.node.Children()[0])
	assert.Equal(t, leftUpdated.node, ll.node.Parent())

	tr, err := NewTree(root.node)
	require.NoError(t, err)

	var order []string
	require.NoError(t, tr.DFS(func(n *Node[*wrapper]) (bool, error) {
		order = append(order, n.Value().name)
		return false, nil
	}))
	assert.Equal(t, []string{"root", "left*", "ll", "right"}, order)
}

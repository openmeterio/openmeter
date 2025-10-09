package treex

type Tree[T any] struct {
	root *Node[T]
}

// NewTree attempts to create a new Tree from a root node.
// It traverses the children graph to ensure it is acyclic.
func NewTree[T any](root *Node[T]) (*Tree[T], error) {
	if root == nil {
		return nil, ErrRootNodeIsNil
	}

	visited := make(map[*Node[T]]bool) // visited nodes tracker (as we don't yet know if it's a tree or not)
	onStack := make(map[*Node[T]]bool) // to verify graph is acyclic

	var dfs func(n *Node[T]) error
	dfs = func(n *Node[T]) error {
		if n == nil {
			return ErrNodeGraphInvalid
		}

		if onStack[n] {
			return ErrGraphHasCycle
		}
		if visited[n] {
			return nil
		}

		visited[n] = true
		onStack[n] = true
		for _, child := range n.children {
			if child == nil {
				return ErrNodeGraphInvalid
			}
			if err := dfs(child); err != nil {
				return err
			}
		}
		onStack[n] = false
		return nil
	}

	if err := dfs(root); err != nil {
		return nil, err
	}

	return &Tree[T]{root: root}, nil
}

func (t *Tree[T]) Root() *Node[T] {
	return t.root
}

func (t *Tree[T]) DFS(cb func(n *Node[T]) (stop bool, err error)) error {
	if t.root == nil {
		return nil
	}

	var walk func(n *Node[T]) error
	walk = func(n *Node[T]) error {
		stop, err := cb(n)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}

		for _, child := range n.children {
			if child == nil {
				return ErrNodeGraphInvalid
			}
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}

	return walk(t.root)
}

// Leafs returns all leaf nodes in the tree
func (t *Tree[T]) Leafs() []*Node[T] {
	leafs := make([]*Node[T], 0)

	_ = t.DFS(func(n *Node[T]) (bool, error) {
		if n.IsLeaf() {
			leafs = append(leafs, n)
		}
		return false, nil
	})

	return leafs
}

// SwapNode can swap any node in the tree including the root.
// If the old node wasn't found an error is returned.
// The new node's parent will be set to the old node's parent.
//
// CAUTION: if you swap a node while walking the tree, you should start backtracking after the swap
// otherwise you'll keep iterating the detached subtree
func (t *Tree[T]) SwapNode(old *Node[T], new *Node[T]) error {
	if old == t.Root() {
		t.root = new
		return nil
	}

	if old.Parent() == nil {
		return ErrNodeHasNoParentButNotRoot
	}

	parent := old.Parent()
	return parent.SwapChild(old, new)
}

# treex

<!-- archie:ai-start -->

> Generic pointer-identity tree/node library (Node[T], Tree[T]) used by openmeter/subscription and pkg/models for hierarchical structures with cycle-checked construction and DFS traversal. Nodes track parent/children by pointer identity, enabling immutable-style subtree swaps.

## Patterns

**Node value must be a non-nil pointer** — NewNode[T] panics unless the value's reflect.Kind is Pointer and it is non-nil. This enforces the wrapper-embedding pattern where a struct holds a *Node[*Self] pointing back to itself. (`func NewNode[T any](value T) *Node[T] { if reflect.ValueOf(value).Kind() != reflect.Pointer { panic("Node value has to be a pointer") } ... }`)
**Parent/child links maintained by pointer identity** — AddChild/RemoveChild/SwapChild update both the children slice and the child.parent pointer; lookups use pointer equality (lo.Find c == child). Detached nodes have parent set to nil. (`func (n *Node[T]) AddChild(child *Node[T]) { n.children = append(n.children, child); child.parent = n }`)
**Tree construction validates acyclicity via DFS** — NewTree runs a DFS with visited+onStack maps, returning ErrGraphHasCycle on back-edges, ErrNodeGraphInvalid on nil nodes/children, ErrRootNodeIsNil on nil root. It does NOT clone — the passed root pointer becomes tr.root. (`if onStack[n] { return ErrGraphHasCycle }`)
**Pre-order DFS with prune and error propagation** — Tree.DFS callback returns (stop bool, err error): err aborts traversal immediately; stop prunes the current node's subtree but continues with siblings. Used by Leafs(). (`err = tr.DFS(func(n *Node[*T]) (bool, error) { ...; if cond { return true, nil } /* prune */; return false, nil })`)
**Immutable subtree updates via ShallowClone + SwapChild/SwapNode** — ShallowClone copies the node, keeps the same value/parent, copies the children slice and reattaches children's parent pointers to the clone. Combine with SwapChild (or Tree.SwapNode for the root) to replace a node without mutating the original subtree. (`updated.node = old.node.ShallowClone(); updated.node.SetValue(updated); parent.SwapChild(old.node, updated.node)`)
**Sentinel errors in errors.go** — All structural failures use the exported sentinel errors (ErrRootNodeIsNil, ErrGraphHasCycle, ErrNodeHasNoParentButNotRoot, ErrNodeGraphInvalid) so callers assert with errors.Is rather than string matching. (`assert.ErrorIs(t, err, ErrGraphHasCycle)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `node.go` | Node[T] type plus NewNode, AddChild/RemoveChild/SwapChild, ShallowClone, DeepClone, IsLeaf/IsRoot, Value/SetValue/Parent/Children. | NewNode panics on non-pointer or nil value (the one place treex panics — by design, library-only). ShallowClone reattaches only first-level children; DeepClone recurses and detaches the returned root (nil parent). |
| `tree.go` | Tree[T] wrapper: NewTree (cycle-checked), Root, DFS, Leafs, SwapNode. | SwapNode handles root replacement (sets t.root) but errors with ErrNodeHasNoParentButNotRoot for a parentless non-root. Comment warns: swapping a node mid-DFS-walk requires backtracking or you iterate the detached subtree. |
| `errors.go` | Exported sentinel errors for invalid graph/node structure. | Return these (not ad-hoc fmt.Errorf) for structural failures so errors.Is checks keep working. |
| `usage_example_test.go` | Canonical wrapper-embedding usage (struct owns *Node[*wrapper]) and immutable-update recipes. | This is the intended consumption pattern (mirrors subscription FieldDescriptor); follow it when adding new tree-backed types. |

## Anti-Patterns

- Calling NewNode with a non-pointer or nil pointer value — it panics; always pass a non-nil *T.
- Manually appending to or reordering Node.children without updating child.parent — breaks pointer-identity invariants RemoveChild/SwapChild rely on.
- Assuming NewTree clones or copies the graph — it validates in place and reuses the root pointer; mutating the original after still affects the tree.
- Swapping a node during a DFS walk and continuing forward instead of backtracking — you will traverse the now-detached subtree.
- Asserting structural failures by error string instead of errors.Is against the exported sentinels.

## Decisions

- **Nodes require pointer values and track identity by pointer.** — Consumers (subscription, models) embed a back-pointer (struct holding *Node[*Self]); pointer identity lets a value participate in a larger tree and supports SwapChild lookups by ==.
- **NewTree validates acyclicity but does not deep-copy.** — Trees can be large; copying on every construction would be wasteful, and immutable updates are opt-in via ShallowClone/DeepClone where callers actually need isolation.
- **ShallowClone reattaches first-level children to the clone.** — Enables immutable-style replacement of a single node while preserving the existing subtree without a full deep copy.

## Example: Build a cycle-checked tree of wrapper structs and traverse pre-order

```
import "github.com/openmeterio/openmeter/pkg/treex"

type wrapper struct {
	name string
	node *treex.Node[*wrapper]
}

func newWrapper(name string) *wrapper {
	w := &wrapper{name: name}
	w.node = treex.NewNode(w) // value must be non-nil pointer
	return w
}

root, child := newWrapper("root"), newWrapper("child")
root.node.AddChild(child.node)
// ...
```

<!-- archie:ai-end -->

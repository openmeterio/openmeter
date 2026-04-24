# treex

<!-- archie:ai-start -->

> Generic typed tree data structure (Node[T], Tree[T]) with DFS traversal, cycle detection, and shallow/deep clone operations; used by subscription and product catalog domains to model hierarchical rate-card or field descriptor trees where immutable-style updates via node swapping are required.

## Patterns

**Node values must be non-nil pointers** — NewNode[T](v T) panics if v is not a non-nil pointer. T must be a pointer type (e.g. *MyStruct). This enforces identity semantics for SwapChild/SwapNode lookups which compare by pointer equality. (`node := treex.NewNode(&myStruct{id: 1})`)
**NewTree validates acyclicity at construction time** — Call NewTree(root) to get a validated *Tree — it returns ErrGraphHasCycle if the children graph has a cycle and ErrNodeGraphInvalid for nil children. Build the full node graph first, then wrap once with NewTree. (`tr, err := treex.NewTree(rootNode); if errors.Is(err, treex.ErrGraphHasCycle) { ... }`)
**ShallowClone for immutable-style node updates** — ShallowClone copies the node struct, copies the children slice, and reattaches existing children to the clone. Use this to 'rename' or 'update' a node value without rebuilding the subtree; then call SwapNode/SwapChild on the parent. (`updated := &wrapper{name: "new"}; updated.node = old.node.ShallowClone(); updated.node.SetValue(updated); root.node.SwapChild(old.node, updated.node)`)
**DFS returns (stop bool, err error) — stop prunes the subtree, not the whole walk** — Returning stop=true from the DFS callback halts descent into that node's children but sibling subtrees continue. Returning a non-nil error aborts the entire walk. (`tr.DFS(func(n *treex.Node[*T]) (bool, error) { if skip(n) { return true, nil }; return false, process(n) })`)
**Self-referential node embedding pattern** — Domain structs that participate in a tree embed a *Node[*Self] field and set it in a constructor. This is the canonical usage pattern shown in usage_example_test.go. (`type FieldDescriptor struct { name string; node *treex.Node[*FieldDescriptor] }
func newFD(name string) *FieldDescriptor { fd := &FieldDescriptor{name: name}; fd.node = treex.NewNode(fd); return fd }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `node.go` | Node[T] struct with parent/children management; ShallowClone and DeepClone for copy semantics | SwapChild and RemoveChild use pointer equality (c == child) not value equality — always pass the exact *Node pointer; ShallowClone reattaches existing first-level children to the clone which mutates child.parent |
| `tree.go` | Tree[T] wrapper with DFS, Leafs, and SwapNode; constructs with cycle detection | SwapNode on the root replaces t.root directly without validation — the caller is responsible for ensuring new is non-nil; DFS comment warns: if you swap a node mid-walk, backtrack immediately |
| `errors.go` | Sentinel errors for nil root, cycles, invalid graph structure | Use errors.Is for matching — all three are package-level var errors |

## Anti-Patterns

- Passing value types (non-pointers) to NewNode — panics immediately
- Modifying node.children slice directly instead of using AddChild/RemoveChild/SwapChild — bypasses parent-pointer bookkeeping
- Continuing DFS traversal after SwapNode in the callback without stopping — the callback will follow the detached old subtree

## Decisions

- **Node values are constrained to pointer types enforced at runtime via reflect** — Tree mutations (SwapChild, RemoveChild) rely on pointer identity comparison; value types would make sibling lookup ambiguous
- **Cycle detection runs at NewTree time, not incrementally on AddChild** — Allows building node graphs freely before sealing them into a Tree; the DFS validation is O(n) and only paid once at construction

<!-- archie:ai-end -->

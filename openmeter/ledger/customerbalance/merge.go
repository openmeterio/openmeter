package customerbalance

import (
	"container/heap"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func compareCreditTransactionsByCursor(a, b CreditTransaction) int {
	return creditTransactionCursor(a).Compare(creditTransactionCursor(b))
}

type mergeListState struct {
	items []CreditTransaction
	next  int
}

type mergeHeapNode struct {
	listIndex int
	item      CreditTransaction
}

type mergeHeap struct {
	nodes []mergeHeapNode
	cmp   func(a, b CreditTransaction) int
}

func (h mergeHeap) Len() int {
	return len(h.nodes)
}

func (h mergeHeap) Less(i, j int) bool {
	return h.cmp(h.nodes[i].item, h.nodes[j].item) > 0
}

func (h mergeHeap) Swap(i, j int) {
	h.nodes[i], h.nodes[j] = h.nodes[j], h.nodes[i]
}

func (h *mergeHeap) Push(x any) {
	h.nodes = append(h.nodes, x.(mergeHeapNode))
}

func (h *mergeHeap) Pop() any {
	old := h.nodes
	last := len(old) - 1
	node := old[last]
	h.nodes = old[:last]
	return node
}

func mergeSortedLists(lists [][]CreditTransaction, limit int, cmp func(a, b CreditTransaction) int) ([]CreditTransaction, bool) {
	if limit <= 0 {
		return []CreditTransaction{}, false
	}

	states := make([]mergeListState, len(lists))
	mergeQ := &mergeHeap{
		nodes: make([]mergeHeapNode, 0, len(lists)),
		cmp:   cmp,
	}

	for i, items := range lists {
		if len(items) == 0 {
			continue
		}

		states[i] = mergeListState{
			items: items,
			next:  1,
		}

		heap.Push(mergeQ, mergeHeapNode{
			listIndex: i,
			item:      items[0],
		})
	}

	merged := make([]CreditTransaction, 0, limit)
	for mergeQ.Len() > 0 && len(merged) < limit {
		node := heap.Pop(mergeQ).(mergeHeapNode)
		merged = append(merged, node.item)

		state := &states[node.listIndex]
		if state.next >= len(state.items) {
			continue
		}

		nextItem := state.items[state.next]
		state.next++
		heap.Push(mergeQ, mergeHeapNode{
			listIndex: node.listIndex,
			item:      nextItem,
		})
	}

	return merged, mergeQ.Len() > 0
}

func creditTransactionCursor(tx CreditTransaction) ledger.TransactionCursor {
	return ledger.TransactionCursor{
		BookedAt:  tx.BookedAt,
		CreatedAt: tx.CreatedAt,
		ID:        tx.ID,
	}
}

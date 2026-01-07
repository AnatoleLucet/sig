package internal

import "iter"

type PriorityHeap struct {
	min int
	max int

	nodes []*heapNode // [height]head

	loopkup map[*Computed]*heapNode // for O(1) removal
}

type heapNode struct {
	node *Computed

	next *heapNode
	prev *heapNode
}

func NewHeap() *PriorityHeap {
	return &PriorityHeap{
		min:     0,
		max:     0,
		nodes:   make([]*heapNode, 2000),
		loopkup: make(map[*Computed]*heapNode),
	}
}

func (h *PriorityHeap) Insert(node *Computed) {
	if node.HasFlag(FlagInHeap) {
		return
	}
	node.AddFlag(FlagInHeap)

	entry := &heapNode{node: node}
	h.loopkup[node] = entry

	height := node.GetHeight()

	if h.nodes[height] == nil {
		h.nodes[height] = entry
		entry.prev = entry // loop to self
		entry.next = nil
	} else {
		head := h.nodes[height]
		tail := head.prev

		tail.next = entry
		entry.prev = tail
		entry.next = nil
		head.prev = entry
	}

	if height > h.max {
		h.max = height
	}
}

func (h *PriorityHeap) InsertAll(nodes iter.Seq[*Computed]) {
	for node := range nodes {
		h.Insert(node)
	}
}

func (h *PriorityHeap) Remove(node *Computed) {
	if !node.HasFlag(FlagInHeap) {
		return
	}
	node.RemoveFlag(FlagInHeap)

	entry, ok := h.loopkup[node]
	if !ok {
		return
	}
	delete(h.loopkup, node)

	height := entry.node.GetHeight()

	// single node
	if entry.prev == entry {
		h.nodes[height] = nil
		entry.prev = entry
		entry.next = nil
		return
	}

	// multiple nodes
	head := h.nodes[height]
	if entry == head {
		h.nodes[height] = entry.next
	} else {
		entry.prev.next = entry.next
	}

	next := entry.next
	if next == nil {
		next = head
	}
	next.prev = entry.prev

	entry.prev = entry
	entry.next = nil
}

// Drain processes each entry in topological order with the `process` function leaving the heap empty.
func (h *PriorityHeap) Drain(process func(*Computed)) {
	for h.min = 0; h.min <= h.max; h.min++ {
		entry := h.nodes[h.min]

		for entry != nil {
			h.Remove(entry.node)
			process(entry.node)
			entry = h.nodes[h.min]
		}
	}

	h.max = 0
}

package proto

// minHeap is a priority queue organized by height
type minHeap struct {
	heap []reactiveNode // heap[height] = circular linked list of nodes at that height
	min  int
	max  int
}

func newHeap() *minHeap {
	return &minHeap{
		heap: make([]reactiveNode, 100), // Pre-allocate reasonable size
		min:  0,
		max:  0,
	}
}

// insert adds a node to the heap at its current height
func (h *minHeap) insert(n reactiveNode) {
	nd := n.node()
	if nd.flags.has(FlagInHeap | FlagRecomputing) {
		return
	}

	// Upgrade Check to Dirty when inserting
	if nd.flags.has(FlagCheck) {
		nd.flags.replace(FlagCheck|FlagDirty, FlagDirty|FlagInHeap)
	} else {
		nd.flags.set(FlagInHeap)
	}

	h.insertAtHeight(n)
}

func (h *minHeap) insertAtHeight(n reactiveNode) {
	nd := n.node()
	height := int(nd.height)

	// Grow heap if needed
	for len(h.heap) <= height {
		h.heap = append(h.heap, nil)
	}

	// Insert into circular linked list at this height
	if h.heap[height] == nil {
		h.heap[height] = n
		nd.heapNext = nil
		nd.heapPrev = n
	} else {
		head := h.heap[height]
		tail := head.node().heapPrev

		tail.node().heapNext = n
		nd.heapPrev = tail
		nd.heapNext = nil
		head.node().heapPrev = n
	}

	if height > h.max {
		h.max = height
	}
}

// remove removes a node from the heap
func (h *minHeap) remove(n reactiveNode) {
	nd := n.node()
	if !nd.flags.has(FlagInHeap) {
		return
	}

	nd.flags.clear(FlagInHeap)

	height := int(nd.height)
	if h.heap[height] == nil {
		return
	}

	// Remove from circular linked list
	if nd.heapPrev == n {
		// Only node at this height
		h.heap[height] = nil
	} else {
		if h.heap[height] == n {
			h.heap[height] = nd.heapNext
		}
		if nd.heapPrev != nil {
			nd.heapPrev.node().heapNext = nd.heapNext
		}
		if nd.heapNext != nil {
			nd.heapNext.node().heapPrev = nd.heapPrev
		} else {
			// We're the tail, update head's prev
			h.heap[height].node().heapPrev = nd.heapPrev
		}
	}

	nd.heapNext = nil
	nd.heapPrev = n
}

// adjustHeight recalculates a node's height based on its dependencies
func (h *minHeap) adjustHeight(n reactiveNode, s *Scheduler) {
	nd := n.node()
	h.remove(n)

	newHeight := uint32(0)
	for l := nd.deps; l != nil; l = l.nextDep {
		depNode := l.dep.node()
		if depHeight := depNode.height; depHeight >= newHeight {
			newHeight = depHeight + 1
		}
	}

	if nd.height != newHeight {
		nd.height = newHeight

		// Propagate height change to subscribers
		for l := nd.subs; l != nil; l = l.nextSub {
			h.insertAtHeight(l.sub)
		}
	}
}

package sigv2

// heapNode represents a node that can be in the execution heap
type heapNode interface {
	Computable
	getNextHeap() heapNode
	setNextHeap(heapNode)
	getPrevHeap() heapNode
	setPrevHeap(heapNode)
}

// heap represents a priority queue ordered by node height
type heap struct {
	heap   []heapNode
	marked bool
	min    int
	max    int
}

// newHeap creates a new heap with given initial capacity
func newHeap(capacity int) *heap {
	return &heap{
		heap:   make([]heapNode, capacity),
		marked: false,
		min:    0,
		max:    0,
	}
}

// actualInsertIntoHeap inserts a node into the heap at its height level
func actualInsertIntoHeap(n heapNode, h *heap) {
	height := n.getHeight()

	// Grow heap if needed
	for height >= len(h.heap) {
		h.heap = append(h.heap, make([]heapNode, 1000)...)
	}

	heapAtHeight := h.heap[height]
	if heapAtHeight == nil {
		h.heap[height] = n
		n.setPrevHeap(n)
		n.setNextHeap(nil)
	} else {
		// Insert at end of circular list
		tail := heapAtHeight.getPrevHeap()
		tail.setNextHeap(n)
		n.setPrevHeap(tail)
		n.setNextHeap(nil)
		heapAtHeight.setPrevHeap(n)
	}

	if height > h.max {
		h.max = height
	}
}

// insertIntoHeap marks a node dirty and inserts it into the heap
func insertIntoHeap(n heapNode, h *heap) {
	flags := n.getFlags()

	// Already in heap or recomputing
	if flags&(FlagInHeap|FlagRecomputingDeps) != 0 {
		return
	}

	// Mark as dirty if currently only check
	if flags&FlagCheck != 0 {
		n.setFlags((flags &^ (FlagCheck | FlagDirty)) | FlagDirty | FlagInHeap)
	} else {
		n.setFlags(flags | FlagInHeap)
	}

	// Don't insert twice if already in heap for height adjustment
	if flags&FlagInHeapHeight == 0 {
		actualInsertIntoHeap(n, h)
	}
}

// insertIntoHeapHeight inserts a node for height adjustment only
func insertIntoHeapHeight(n heapNode, h *heap) {
	flags := n.getFlags()

	if flags&(FlagInHeap|FlagRecomputingDeps|FlagInHeapHeight) != 0 {
		return
	}

	n.setFlags(flags | FlagInHeapHeight)
	actualInsertIntoHeap(n, h)
}

// deleteFromHeap removes a node from the heap
func deleteFromHeap(n heapNode, h *heap) {
	flags := n.getFlags()

	if flags&(FlagInHeap|FlagInHeapHeight) == 0 {
		return
	}

	n.setFlags(flags &^ (FlagInHeap | FlagInHeapHeight))

	height := n.getHeight()
	if height >= len(h.heap) {
		return
	}

	// Single node in circular list
	if n.getPrevHeap() == n {
		h.heap[height] = nil
	} else {
		next := n.getNextHeap()
		dhh := h.heap[height]

		var end heapNode
		if next != nil {
			end = next
		} else {
			end = dhh
		}

		// Update head if we're removing it
		if n == dhh {
			h.heap[height] = next
		} else {
			prev := n.getPrevHeap()
			prev.setNextHeap(next)
		}

		end.setPrevHeap(n.getPrevHeap())
	}

	// Reset node's heap pointers
	n.setPrevHeap(n)
	n.setNextHeap(nil)
}

// markHeap marks all nodes in the heap as needing updates
func markHeap(h *heap, markFn func(heapNode)) {
	if h.marked {
		return
	}

	h.marked = true

	for i := 0; i <= h.max && i < len(h.heap); i++ {
		el := h.heap[i]
		for el != nil {
			if el.getFlags()&FlagInHeap != 0 {
				markFn(el)
			}
			el = el.getNextHeap()
		}
	}
}

// runHeap processes all nodes in the heap in height order
func runHeap(h *heap, recomputeFn func(heapNode)) {
	h.marked = false

	for h.min = 0; h.min <= h.max && h.min < len(h.heap); h.min++ {
		for {
			el := h.heap[h.min]
			if el == nil {
				break
			}

			if el.getFlags()&FlagInHeap != 0 {
				recomputeFn(el)
			} else {
				adjustHeight(el, h)
			}
		}
	}

	h.max = 0
}

// adjustHeight recalculates a node's height based on its dependencies
func adjustHeight(el heapNode, h *heap) {
	deleteFromHeap(el, h)

	newHeight := el.getHeight()

	// Find maximum dependency height
	for d := el.getDeps(); d != nil; d = d.nextDep {
		dep := d.dep

		// Get the actual computed node if this is a signal with owner
		if computable, ok := dep.(Computable); ok {
			depHeight := computable.getHeight()
			if depHeight >= newHeight {
				newHeight = depHeight + 1
			}
		}
	}

	// If height changed, propagate to subscribers
	if el.getHeight() != newHeight {
		el.setHeight(newHeight)

		// Get subscribers from the observable interface
		if obs, ok := interface{}(el).(Observable); ok {
			for s := obs.getSubs(); s != nil; s = s.nextSub {
				if hn, ok := s.sub.(heapNode); ok {
					insertIntoHeapHeight(hn, h)
				}
			}
		}
	}
}

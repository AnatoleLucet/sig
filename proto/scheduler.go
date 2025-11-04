package proto

import "sync"

// Scheduler manages the reactive graph execution
type Scheduler struct {
	mu sync.RWMutex

	clock        uint64
	dirty        *minHeap
	pendingNodes []reactiveNode

	// Batching support
	batchDepth int

	// Tracking active reaction for dependency collection
	activeReaction Reaction
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		dirty:        newHeap(),
		pendingNodes: make([]reactiveNode, 0),
	}
}

// Stabilize processes all dirty nodes and commits pending values
func (s *Scheduler) Stabilize() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runHeap()

	// Commit all pending values atomically
	for _, n := range s.pendingNodes {
		switch node := n.(type) {
		case *memo:
			if node.pendingValue != nil {
				node.value = *node.pendingValue
				node.pendingValue = nil
			}
		case *signal:
			if node.pendingValue != nil {
				node.value = *node.pendingValue
				node.pendingValue = nil
			}
		}
	}
	s.pendingNodes = s.pendingNodes[:0]

	s.clock++
}

// runHeap processes the dirty heap from lowest to highest height
func (s *Scheduler) runHeap() {
	for s.dirty.min = 0; s.dirty.min <= s.dirty.max; s.dirty.min++ {
		for s.dirty.heap[s.dirty.min] != nil {
			n := s.dirty.heap[s.dirty.min]
			nd := n.node()

			if nd.flags.has(FlagInHeap) {
				// This is a dirty node, recompute it
				n.(Reaction).execute(s)
			} else {
				// Just adjusting height
				s.dirty.adjustHeight(n, s)
			}
		}
	}
	s.dirty.max = 0
}

// updateIfNecessary checks if a node needs updating and updates if necessary
func (s *Scheduler) updateIfNecessary(n reactiveNode) {
	nd := n.node()

	// If only marked Check, verify dependencies first
	if nd.flags.has(FlagCheck) {
		for l := nd.deps; l != nil; l = l.nextDep {
			if depReaction, ok := l.dep.(Reaction); ok {
				s.updateIfNecessary(depReaction)
			}
			if nd.flags.has(FlagDirty) {
				break // Already dirty, no need to check more
			}
		}
	}

	// Now recompute if truly dirty
	if nd.flags.has(FlagDirty) {
		if reaction, ok := n.(Reaction); ok {
			reaction.execute(s)
		}
	}

	nd.flags = FlagNone
}

// markNode marks a node and its subscribers as needing checking/recomputation
func (s *Scheduler) markNode(n reactiveNode, state flags) {
	nd := n.node()

	// If already marked with a stronger state, skip
	if (nd.flags & (FlagCheck | FlagDirty)) >= state {
		return
	}

	nd.flags.replace(FlagCheck|FlagDirty, state)

	// Recursively mark all subscribers with Check
	for l := nd.subs; l != nil; l = l.nextSub {
		s.markNode(l.sub, FlagCheck)
	}
}

// Batch executes a function while batching updates
func (s *Scheduler) Batch(fn func()) {
	s.mu.Lock()
	s.batchDepth++
	s.mu.Unlock()

	fn()

	s.mu.Lock()
	s.batchDepth--
	shouldStabilize := s.batchDepth == 0
	s.mu.Unlock()

	if shouldStabilize {
		s.Stabilize()
	}
}

// Global default scheduler
var defaultScheduler = NewScheduler()

// Batch batches updates using the default scheduler
func Batch(fn func()) {
	defaultScheduler.Batch(fn)
}

// Stabilize stabilizes the default scheduler
func Stabilize() {
	defaultScheduler.Stabilize()
}

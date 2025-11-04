package proto

// nodeData contains the scheduling metadata for reactive nodes
type nodeData struct {
	height uint32 // Dependency depth (0 for signals, 1+ for computed)
	flags  flags  // Current state flags

	// Dependency tracking (what this node depends on)
	deps     *link
	depsTail *link

	// Subscription tracking (what depends on this node)
	subs     *link
	subsTail *link

	// Heap management (for circular linked list at each height)
	heapNext reactiveNode
	heapPrev reactiveNode
}

// reactiveNode is the interface that all reactive nodes implement
type reactiveNode interface {
	node() *nodeData
}

// Observable represents something that can be observed (signals, memos)
type Observable interface {
	reactiveNode
	read(s *Scheduler) any
}

// Reaction represents something that reacts to changes (effects, memos)
type Reaction interface {
	reactiveNode
	execute(s *Scheduler)
}

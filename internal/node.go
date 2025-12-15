package internal

type NodeFlags int

const (
	FlagNone   NodeFlags = 0
	FlagInHeap NodeFlags = 1 << 0 // node is currently in heap
)

type ReactiveNode struct {
	// the node's state
	flags NodeFlags

	// the current height of the node in the priority graph
	height int

	// the clock tick at which the node was last updated
	version Tick
}

func (r *Runtime) NewNode() *ReactiveNode {
	return &ReactiveNode{}
}

// HasFlag checks if the given flag is set
func (n *ReactiveNode) HasFlag(flag NodeFlags) bool {
	return n.flags&flag != 0
}

// AddFlag adds the given flag
func (n *ReactiveNode) AddFlag(flag NodeFlags) {
	n.flags |= flag
}

// RemoveFlag removes the given flag
func (n *ReactiveNode) RemoveFlag(flag NodeFlags) {
	n.flags &^= flag
}

// SetFlags sets the flags to exact value
func (n *ReactiveNode) SetFlags(flags NodeFlags) {
	n.flags = flags
}

func (n *ReactiveNode) SetVersion(t Tick) {
	n.version = t
}

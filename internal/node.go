package internal

type NodeFlags int

const (
	FlagNone     NodeFlags = 0
	FlagCheck    NodeFlags = 1 << 0 // node needs to be checked for updates
	FlagDirty    NodeFlags = 2 << 0 // node is dirty and needs to be recomputed
	FlagInHeap   NodeFlags = 3 << 0 // node is currently in heap for update scheduling
	FlagDisposed NodeFlags = 4 << 0 // node has been disposed
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

package internal

import "sync"

type NodeFlags int

const (
	FlagNone     NodeFlags = 0
	FlagCheck    NodeFlags = 1 << 0 // node needs to be checked for updates
	FlagDirty    NodeFlags = 1 << 1 // node is dirty and needs to be recomputed
	FlagInHeap   NodeFlags = 1 << 2 // node is currently in heap for update scheduling
	FlagDisposed NodeFlags = 1 << 3 // node has been disposed
)

type ReactiveNode struct {
	mu sync.RWMutex

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
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.flags&flag != 0
}

// AddFlag adds the given flag
func (n *ReactiveNode) AddFlag(flag NodeFlags) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.flags |= flag
}

// RemoveFlag removes the given flag
func (n *ReactiveNode) RemoveFlag(flag NodeFlags) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.flags &^= flag
}

// SetFlags sets the flags to exact value
func (n *ReactiveNode) SetFlags(flags NodeFlags) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.flags = flags
}

func (n *ReactiveNode) SetVersion(t Tick) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.version = t
}

func (n *ReactiveNode) GetHeight() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.height
}

func (n *ReactiveNode) SetHeight(h int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.height = h
}
